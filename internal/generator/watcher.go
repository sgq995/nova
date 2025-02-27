package generator

import (
	"context"
	"io/fs"
	"log"
	"path/filepath"
	"time"

	"github.com/sgq995/nova/internal/project"
)

type Watcher struct {
	ctx      context.Context
	pattern  string
	onChange func(string)
	fileInfo map[string]time.Time
}

func NewWatcher(ctx context.Context, pattern string, onChange func(string)) *Watcher {
	return &Watcher{
		ctx:      ctx,
		pattern:  pattern,
		onChange: onChange,
		fileInfo: make(map[string]time.Time),
	}
}

func (w *Watcher) Watch(dir string) {
	target := project.Abs(dir)

	for {
		err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			filename := filepath.Base(path)
			if matched, err := filepath.Match(w.pattern, filename); !matched {
				return err
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			modTime := info.ModTime()
			if lastModTime, exists := w.fileInfo[path]; exists && modTime.After(lastModTime) {
				w.onChange(path)
			}
			w.fileInfo[path] = modTime

			return nil
		})

		if err != nil {
			log.Fatalln(err)
		}

		time.Sleep(1 * time.Second)

		select {
		case <-w.ctx.Done():
			return

		default:
		}
	}
}
