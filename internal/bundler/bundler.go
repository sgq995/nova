package bundler

import (
	_ "embed"
	"io/fs"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"segoqu.com/nova/internal/project"
)

//go:embed hmr.js
var hmr string

func findFiles(dir string) ([]string, error) {
	validExtensions := map[string]bool{
		".js":  true,
		".jsx": true,
		".ts":  true,
		".tsx": true,
		".css": true,
	}

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
	entries, err := findFiles("src/pages")
	if err != nil {
		return nil, err
	}

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
