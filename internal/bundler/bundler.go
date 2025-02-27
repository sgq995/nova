package bundler

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sgq995/nova/internal/project"
	"github.com/tdewolff/minify/v2"
	minifyHtml "github.com/tdewolff/minify/v2/html"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

//go:embed hmr.js
var hmr string

func findFiles(dir string, validExtensions map[string]bool) ([]string, error) {
	var files []string
	target := project.Abs(dir)
	err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if !validExtensions[ext] {
			return nil
		}

		files = append(files, path)

		return nil
	})

	return files, err
}

func cleanDir(dir string) error {
	err := os.RemoveAll(dir)
	if err != nil {
		return err
	}
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}
	return nil
}

func Development(dir string) (api.BuildContext, error) {
	// TODO: run on init
	hmrResult := api.Build(api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents: hmr,
		},
		Bundle:            true,
		Format:            api.FormatESModule,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		Sourcemap:         api.SourceMapNone,
	})
	if len(hmrResult.Errors) > 0 {
		panic(hmrResult.Errors)
	}
	hmrBundle := string(hmrResult.OutputFiles[0].Contents)

	validExtensions := map[string]bool{
		".js":  true,
		".jsx": true,
		".ts":  true,
		".tsx": true,
		".css": true,
	}
	entries, err := findFiles(dir, validExtensions)
	if err != nil {
		return nil, err
	}

	outDir := project.Abs(filepath.Join(".nova", "static"))
	ctx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: entries,
		Bundle:      true,
		Format:      api.FormatESModule,
		Splitting:   true,
		Outdir:      outDir,
		Sourcemap:   api.SourceMapLinked,
		Banner: map[string]string{
			"js": hmrBundle,
		},
	})
	if ctxErr != nil {
		return nil, ctxErr
	}

	return ctx, nil
}

type buildContext struct {
	static    api.BuildContext
	templates api.BuildContext
}

func (ctx *buildContext) Cancel() {
	ctx.static.Cancel()
	ctx.templates.Cancel()
}

func (ctx *buildContext) Dispose() {
	ctx.static.Dispose()
	ctx.templates.Dispose()
}

func (ctx *buildContext) Rebuild() api.BuildResult {
	r1 := ctx.static.Rebuild()
	r2 := ctx.templates.Rebuild()
	return api.BuildResult{
		Errors:      slices.Concat(r1.Errors, r2.Errors),
		Warnings:    slices.Concat(r1.Warnings, r2.Warnings),
		Metafile:    r1.Metafile,
		OutputFiles: slices.Concat(r1.OutputFiles, r2.OutputFiles),
		MangleCache: r1.MangleCache,
	}
}

func (ctx *buildContext) Serve(api.ServeOptions) (api.ServeResult, error) {
	return api.ServeResult{}, errors.ErrUnsupported
}

func (ctx *buildContext) Watch(api.WatchOptions) error {
	return errors.ErrUnsupported
}

func Production(dir string) (api.BuildContext, error) {
	static, err := findFiles(dir, map[string]bool{
		".js":  true,
		".jsx": true,
		".ts":  true,
		".tsx": true,
		".css": true,
	})
	if err != nil {
		return nil, err
	}

	templates, err := findFiles(dir, map[string]bool{
		".html": true,
	})
	if err != nil {
		return nil, err
	}

	outDir := project.Abs(".nova")
	staticDir := project.Abs(filepath.Join(".nova", "static"))
	err = cleanDir(staticDir)
	if err != nil {
		return nil, err
	}
	templatesDir := project.Abs(filepath.Join(".nova", "templates"))
	err = cleanDir(templatesDir)
	if err != nil {
		return nil, err
	}

	// TODO: independent context for html files only to save html files without hash
	staticCtx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: static,

		EntryNames: "[dir]/[name].[hash]",

		Bundle: true,
		Write:  true,

		Metafile: true,

		Format:    api.FormatESModule,
		Splitting: true,

		Outdir: staticDir,

		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,

		Sourcemap: api.SourceMapNone,

		Plugins: []api.Plugin{
			{
				Name: "nova-manifest",
				Setup: func(pb api.PluginBuild) {
					pb.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
						os.WriteFile(filepath.Join(outDir, "meta.json"), []byte(result.Metafile), 0755)
						return api.OnEndResult{}, nil
					})
				},
			},
		},
	})
	if ctxErr != nil {
		return nil, ctxErr
	}

	tempaltesCtx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: templates,

		EntryNames: "[dir]/[name]",

		Write: true,

		Outdir: templatesDir,

		Sourcemap: api.SourceMapNone,

		Plugins: []api.Plugin{
			{
				Name: "nova-html",
				Setup: func(pb api.PluginBuild) {
					pb.OnLoad(api.OnLoadOptions{Filter: "\\.html$"}, func(ola api.OnLoadArgs) (api.OnLoadResult, error) {
						b, err := os.ReadFile(ola.Path)
						if err != nil {
							return api.OnLoadResult{}, err
						}

						metafile, err := os.ReadFile(filepath.Join(outDir, "meta.json"))
						if err != nil {
							return api.OnLoadResult{}, err
						}

						meta := make(map[string]any)
						err = json.Unmarshal(metafile, &meta)
						if err != nil {
							return api.OnLoadResult{}, err
						}

						srcMap := make(map[string]string)
						outputs := meta["outputs"].(map[string]any)
						// outPrefix := path.Join(".nova", "static")
						outPrefix := ".nova"
						inPrefix := filepath.ToSlash(dir)
						for out := range outputs {
							val := outputs[out].(map[string]any)
							in := val["entryPoint"].(string)

							outSrc := strings.TrimPrefix(out, outPrefix)
							inSrc := strings.TrimPrefix(in, inPrefix)

							srcMap[inSrc] = outSrc
						}

						currentDir := strings.TrimPrefix(ola.Path, filepath.Join(project.Root(), dir))
						currentDir = filepath.Dir(currentDir)
						currentDir = filepath.ToSlash(currentDir)

						doc, err := html.Parse(bytes.NewReader(b))
						mapFiles := func(n *html.Node) {
							if n.Type != html.ElementNode {
								return
							}

							if n.DataAtom != atom.Script && n.DataAtom != atom.Link {
								return
							}

							target := "src"
							if n.DataAtom == atom.Link {
								target = "href"
							}

							for i, attr := range n.Attr {
								if attr.Key != target {
									continue
								}

								in := path.Clean(attr.Val)
								in = path.Join(currentDir, in)

								if hashed, ok := srcMap[in]; ok {
									newSrc := hashed
									n.Attr[i].Val = newSrc
								}
							}
						}

						for n := range doc.Descendants() {
							mapFiles(n)
						}

						var buf bytes.Buffer
						html.Render(&buf, doc)

						m := minify.New()
						m.Add("text/html", &minifyHtml.Minifier{TemplateDelims: minifyHtml.GoTemplateDelims})
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

					pb.OnEnd(func(result *api.BuildResult) (api.OnEndResult, error) {
						// TODO: convert scripts and links to respective css and js
						return api.OnEndResult{}, nil
					})
				},
			},
		},
	})
	if ctxErr != nil {
		return nil, ctxErr
	}

	return &buildContext{static: staticCtx, templates: tempaltesCtx}, nil
}
