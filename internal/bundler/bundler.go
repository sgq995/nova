package bundler

import (
	"io/fs"
	"path/filepath"

	"github.com/evanw/esbuild/pkg/api"
	"segoqu.com/nova/internal/project"
)

func findFiles(dir string) ([]string, error) {
	validExtensions := map[string]bool{
		".js":  true,
		".jsx": true,
		".ts":  true,
		".tsx": true,
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

	outDir := project.Abs(filepath.Join(".nova", "static"))
	ctx, ctxErr := api.Context(api.BuildOptions{
		EntryPoints: entries,
		Bundle:      true,
		Format:      api.FormatESModule,
		Splitting:   true,
		Outdir:      outDir,
		Sourcemap:   api.SourceMapInline,
	})
	if ctxErr != nil {
		return nil, ctxErr
	}

	return ctx, nil
}
