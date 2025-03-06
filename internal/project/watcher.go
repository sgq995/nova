package project

import (
	"context"
	"io/fs"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/sgq995/nova/internal/module"
	"github.com/sgq995/nova/internal/utils"
)

type watcher struct {
	ctx       context.Context
	callbacks map[string]func(string)
	fileInfo  map[string]time.Time
}

func newWatcher(ctx context.Context, callbacks map[string]func(string)) *watcher {
	return &watcher{
		ctx:       ctx,
		callbacks: callbacks,
		fileInfo:  make(map[string]time.Time),
	}
}

func (w *watcher) watch(dir string) {
	target := module.Abs(dir)

	for {
		err := filepath.WalkDir(target, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			name := filepath.Base(path)
			matches := []string{}
			for matcher := range w.callbacks {
				patterns := strings.Split(matcher, ",")
				for _, pattern := range patterns {
					if matched := utils.Must(filepath.Match(pattern, name)); matched {
						matches = append(matches, matcher)
					}
				}
			}

			if len(matches) == 0 {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			modTime := info.ModTime()
			if lastModTime, exists := w.fileInfo[path]; exists && modTime.After(lastModTime) {
				for _, matcher := range matches {
					cb := w.callbacks[matcher]
					cb(path)
				}
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
