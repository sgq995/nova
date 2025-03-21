package esbuild

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/utils"
	"github.com/tdewolff/minify/v2"
	minifyHTML "github.com/tdewolff/minify/v2/html"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func esbuildError(messages []api.Message) error {
	errs := []error{}
	for _, msg := range messages {
		text := fmt.Sprintf("%s:%d: %s", msg.Location.File, msg.Location.Line, msg.Text)
		errs = append(errs, errors.New(text))
	}
	return errors.Join(errs...)
}

type ESBuildContext interface {
	Build() (map[string][]byte, error)

	Dispose()
}

type esbuildContextImpl struct {
	app         api.BuildContext
	nodeModules api.BuildContext
}

func (ctx *esbuildContextImpl) Build() (map[string][]byte, error) {
	result := ctx.nodeModules.Rebuild()
	if len(result.Errors) > 0 {
		return nil, esbuildError(result.Errors)
	}

	result = ctx.app.Rebuild()
	if len(result.Errors) > 0 {
		return nil, esbuildError(result.Errors)
	}

	files := map[string][]byte{}
	for _, f := range result.OutputFiles {
		files[f.Path] = f.Contents
	}

	return files, nil
}

func (ctx *esbuildContextImpl) Dispose() {
	ctx.nodeModules.Dispose()
	ctx.app.Dispose()
}

type ESBuild struct {
	config *config.Config
}

func NewESBuild(c *config.Config) *ESBuild {
	return &ESBuild{config: c}
}

func (esbuild *ESBuild) Context(entryPoints []string) ESBuildContext {
	outDir := module.Join(esbuild.config.Codegen.OutDir, "static")

	nodeModules := map[string]string{}
	appCtx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: entryPoints,
		Outdir:      outDir,
		Format:      api.FormatESModule,
		Bundle:      true,
		Splitting:   true,
		Sourcemap:   api.SourceMapInline,
		Banner: map[string]string{
			"js": `import "/@nova/hmr.js";`,
		},
		Plugins: []api.Plugin{
			{
				Name: "nova-node_modules",
				Setup: func(pb api.PluginBuild) {
					pb.OnResolve(api.OnResolveOptions{Filter: "^[^./]"}, func(ora api.OnResolveArgs) (api.OnResolveResult, error) {
						packagepath := module.Join("node_modules", ora.Path)
						_, err := os.Stat(packagepath)
						switch {
						case err == os.ErrNotExist:
							return api.OnResolveResult{}, nil

						case err != nil:
							return api.OnResolveResult{}, err

						default:
						}

						if strings.HasSuffix(ora.ResolveDir, module.Abs("node_modules")) {
							return api.OnResolveResult{}, nil
						}

						logger.Debugf("[esbuild] resolve (%s)", ora.Path)

						result := pb.Resolve(ora.Path, api.ResolveOptions{
							ResolveDir: module.Abs("node_modules"),
							Importer:   ora.Importer,
							Namespace:  ora.Namespace,
							PluginData: ora.PluginData,
							With:       ora.With,
							Kind:       ora.Kind,
						})
						nodeModules[ora.Path] = result.Path

						return api.OnResolveResult{
							External:  true,
							Path:      "/@node_modules/" + ora.Path + ".js",
							Namespace: "@node_modules",

							PluginData: result.PluginData,
							Errors:     result.Errors,
							Warnings:   result.Warnings,
							Suffix:     result.Suffix,
						}, nil
					})
				},
			},
		},
	})
	if ctxErr != nil {
		logger.Errorf("%+v", ctxErr)
	}

	result := appCtx.Rebuild()
	if len(result.Errors) > 0 {
		logger.Errorf("%+v", esbuildError(result.Errors))
	}

	nodeModulesEntries := []api.EntryPoint{}
	for dep, file := range nodeModules {
		nodeModulesEntries = append(nodeModulesEntries, api.EntryPoint{
			InputPath:  file,
			OutputPath: dep,
		})
	}
	nodeModulesCtx, ctxErr := api.Context(api.BuildOptions{
		EntryPointsAdvanced: nodeModulesEntries,
		Bundle:              true,
		Write:               true,
		Outdir:              module.Join("node_modules", ".nova"),
		Format:              api.FormatESModule,
		MinifyWhitespace:    true,
		MinifyIdentifiers:   true,
		MinifySyntax:        true,
	})
	if ctxErr != nil {
		logger.Errorf("%+v", ctxErr)
	}

	result = nodeModulesCtx.Rebuild()
	if len(result.Errors) > 0 {
		logger.Errorf("%+v", esbuildError(result.Errors))
	}

	return &esbuildContextImpl{
		app:         appCtx,
		nodeModules: nodeModulesCtx,
	}
}

type BuildOptions struct {
	EntryPoints []string
	Outdir      string
	EntryMap    map[string]string
	Hashing     bool
}

func (esbuild *ESBuild) Build(options BuildOptions) (map[string]string, error) {
	pages := module.Abs(esbuild.config.Router.Pages)
	entryNames := "[dir]/[name].[hash]"
	if !options.Hashing {
		entryNames = "[dir]/[name]"
	}

	utils.Clean(options.Outdir)
	result := api.Build(api.BuildOptions{
		EntryPoints:       options.EntryPoints,
		EntryNames:        entryNames,
		Bundle:            true,
		Write:             true,
		Metafile:          true,
		Format:            api.FormatESModule,
		Splitting:         true,
		Outdir:            options.Outdir,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Sourcemap:         api.SourceMapNone,
		LegalComments:     api.LegalCommentsExternal,
		Plugins: []api.Plugin{
			{
				Name: "nova-metafile",
				Setup: func(pb api.PluginBuild) {
					pb.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
						set := filepath.Base(options.Outdir)
						name := strings.Join([]string{set, "metafile.json"}, ".")
						err := os.WriteFile(module.Join(esbuild.config.Codegen.OutDir, name), []byte(result.Metafile), 0755)
						return api.OnEndResult{}, err
					})
				},
			},
			{
				Name: "nova-html",
				Setup: func(pb api.PluginBuild) {
					pb.OnLoad(api.OnLoadOptions{Filter: "\\.html$"}, func(ola api.OnLoadArgs) (api.OnLoadResult, error) {
						b, err := os.ReadFile(ola.Path)
						if err != nil {
							return api.OnLoadResult{}, err
						}

						current := filepath.Dir(ola.Path)

						doc, err := html.Parse(bytes.NewReader(b))
						if err != nil {
							return api.OnLoadResult{}, err
						}

						for n := range doc.Descendants() {
							if n.Type != html.ElementNode {
								continue
							}

							var index int
							switch n.DataAtom {
							case atom.Script:
								index = slices.IndexFunc(n.Attr, func(attr html.Attribute) bool {
									return attr.Key == "src"
								})

							case atom.Link:
								index = slices.IndexFunc(n.Attr, func(attr html.Attribute) bool {
									return attr.Key == "href"
								})

							default:
								continue
							}
							if index == -1 {
								continue
							}

							var filename string
							name := n.Attr[index].Val
							if strings.HasPrefix(name, "/") {
								filename = filepath.Join(pages, name)
							} else {
								filename = filepath.Join(current, name)
							}

							if filename, exists := options.EntryMap[filename]; exists {
								n.Attr[index].Val = "/static/" + filename
							}
						}

						var buf bytes.Buffer
						html.Render(&buf, doc)

						m := minify.New()
						m.AddFunc("text/html", minifyHTML.Minify)
						b, err = m.Bytes("text/html", buf.Bytes())
						if err != nil {
							return api.OnLoadResult{}, err
						}

						contents := string(b)
						return api.OnLoadResult{
							Contents: &contents,
							Loader:   api.LoaderCopy,
						}, nil
					})
				},
			},
		},
	})
	if len(result.Errors) > 0 {
		return nil, esbuildError(result.Errors)
	}

	meta := make(map[string]any)
	err := json.Unmarshal([]byte(result.Metafile), &meta)
	if err != nil {
		return nil, err
	}

	wd, _ := os.Getwd()
	entryMap := make(map[string]string)
	outputs := meta["outputs"].(map[string]any)
	for out := range outputs {
		val := outputs[out].(map[string]any)
		in, ok := val["entryPoint"].(string)
		if !ok {
			continue
		}

		inSrc := filepath.Join(wd, in)
		outSrc := filepath.Join(wd, out)
		outSrc, _ = filepath.Rel(options.Outdir, outSrc)

		entryMap[inSrc] = outSrc
	}

	return entryMap, nil
}
