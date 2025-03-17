package watcher

import (
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

func findFiles(root string) (map[string]time.Time, error) {
	files := map[string]time.Time{}
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		files[path] = info.ModTime()

		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func checkFiles(paths []string) (map[string]time.Time, error) {
	files := map[string]time.Time{}

	for _, filename := range paths {
		info, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		files[filename] = info.ModTime()
	}

	return files, nil
}

type fileEvents struct {
	created []string
	updated []string
	deleted []string
}

func diff(files map[string]time.Time, newFiles map[string]time.Time) *fileEvents {
	created := []string{}
	updated := []string{}
	deleted := []string{}

	for filename, lastTime := range files {
		currentTime, exists := newFiles[filename]
		if exists && currentTime.After(lastTime) {
			updated = append(updated, filename)
		}
		if !exists {
			deleted = append(deleted, filename)
		}
	}

	for filename := range newFiles {
		_, exists := files[filename]
		if !exists {
			created = append(created, filename)
		}
	}

	return &fileEvents{
		created: created,
		updated: updated,
		deleted: deleted,
	}
}
