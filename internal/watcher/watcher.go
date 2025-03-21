package watcher

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/sgq995/nova/internal/module"
)

func WatchDir(ctx context.Context, dir string, callbacks WatchCallbackMap) error {
	root := module.Abs(dir)
	files := map[string]time.Time{}

	// TODO: config scan time
	filesTicker := time.NewTicker(250 * time.Millisecond)
	defer filesTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-filesTicker.C:
			newFiles, err := scanFiles(root)
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
