package watcher

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/module"
)

func dispatchFunc(files []string, ecm eventCallbackMap) {
	for _, filename := range files {
		matches := ecm.match(filename)
		if len(matches) == 0 {
			continue
		}

		for _, cb := range matches {
			// TODO: verify if WaitGroup is needed
			go func() {
				err := cb(filename)
				if err != nil {
					logger.Errorf("%+v", err)
				}
			}()
		}
	}
}

func dispatch(files *fileEvents, callbacks CallbackMap) {
	created, updated, deleted := callbacks.splitEventCallbacks()

	dispatchFunc(files.created, created)

	dispatchFunc(files.updated, updated)

	dispatchFunc(files.deleted, deleted)
}

func WatchDir(ctx context.Context, dir string, callbacks CallbackMap) error {
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
			newFiles, err := findFiles(root)
			if err != nil {
				return err
			}
			events := diff(files, newFiles)
			files = newFiles
			dispatch(events, callbacks)

		default:
			paths := slices.Collect(maps.Keys(files))
			newFiles, err := checkFiles(paths)
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
