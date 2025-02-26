package bundler

import (
	_ "embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/html"
	"segoqu.com/nova/internal/project"
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
	entries, err := findFiles("src/pages", validExtensions)
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

func Production(dir string) (api.BuildContext, error) {
	validExtensions := map[string]bool{
		".js":   true,
		".jsx":  true,
		".ts":   true,
		".tsx":  true,
		".css":  true,
		".html": true,
	}
	entries, err := findFiles("src/pages", validExtensions)
	if err != nil {
		return nil, err
	}

	outDir := project.Abs(".nova")
	staticDir := project.Abs(filepath.Join(".nova", "static"))
	err = os.RemoveAll(staticDir)
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(staticDir, 0755)
	if err != nil {
		return nil, err
	}

	// TODO: independent context for html files only to save html files without hash
	ctx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: entries,

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
			{
				Name: "nova-html",
				Setup: func(pb api.PluginBuild) {
					pb.OnLoad(api.OnLoadOptions{Filter: "\\.html$"}, func(ola api.OnLoadArgs) (api.OnLoadResult, error) {
						b, err := os.ReadFile(ola.Path)
						if err != nil {
							return api.OnLoadResult{}, err
						}

						contents := string(b)

						m := minify.New()
						m.AddFunc("text/html", html.Minify)
						contents, err = m.String("text/html", contents)
						if err != nil {
							return api.OnLoadResult{}, err
						}

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

	return ctx, nil
}
