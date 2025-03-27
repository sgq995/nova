package watcher

import (
	"context"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/sgq995/nova/internal/module"
)

func WatchDir(ctx context.Context, dir string, callbacks CallbackMap) error {
	root := module.Abs(dir)
	files := map[string]time.Time{}

	patterns := []string{}
	for matcher := range callbacks {
		patterns = slices.Concat(patterns, strings.Split(matcher, ","))
	}

	// TODO: config scan time
	filesTicker := time.NewTicker(250 * time.Millisecond)
	defer filesTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-filesTicker.C:
			newFiles, err := scanFiles(root, patterns)
			if err != nil {
				return err
			}
			events := diff(files, newFiles)
			files = newFiles
			dispatch(events, callbacks)

		default:
			paths := slices.Collect(maps.Keys(files))
			newFiles, err := lookupFiles(paths)
			if err != nil {
				return err
			}
			events := diff(files, newFiles)
			files = newFiles
			dispatch(events, callbacks)
			// TODO: config
			time.Sleep(250 * time.Millisecond)
		}
	}
}
