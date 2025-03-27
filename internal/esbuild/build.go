package esbuild

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/sgq995/nova/internal/config"
	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/utils"
	"github.com/tdewolff/minify/v2"
	minifyHTML "github.com/tdewolff/minify/v2/html"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type ESBuild struct {
	config *config.Config
}

func NewESBuild(c *config.Config) *ESBuild {
	return &ESBuild{config: c}
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
