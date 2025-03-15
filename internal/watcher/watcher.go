package watcher

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
	ctx      context.Context
	fileInfo map[string]time.Time
}

func new(ctx context.Context) *watcher {
	return &watcher{
		ctx:      ctx,
		fileInfo: make(map[string]time.Time),
	}
}

func WatchDir(ctx context.Context, dir string, callbacks map[string]func(string)) {
	target := module.Abs(dir)
	fileInfo := make(map[string]time.Time)
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
			for matcher := range callbacks {
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
			if lastModTime, exists := fileInfo[path]; exists && modTime.After(lastModTime) {
				for _, matcher := range matches {
					cb := callbacks[matcher]
					cb(path)
				}
			}
			fileInfo[path] = modTime

			return nil
		})

		if err != nil {
			log.Fatalln(err)
		}

		time.Sleep(1 * time.Second)

		select {
		case <-ctx.Done():
			return

		default:
		}
	}
}
