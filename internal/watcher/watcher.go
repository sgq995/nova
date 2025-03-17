package watcher

import (
	"context"
	"maps"
	"slices"
	"time"

	"github.com/sgq995/nova/internal/module"
)

func dispatchFunc(files []string, match func(string) []Callback, execute func(cb Callback, filename string) error) {
	for _, filename := range files {
		matches := match(filename)
		if len(matches) == 0 {
			continue
		}

		for _, cb := range matches {
			// TODO: verify if WaitGroup is needed
			go execute(cb, filename)
		}
	}
}

func dispatch(files *fileEvents, callbacks CallbackMap) {
	dispatchFunc(files.created, callbacks.match, func(cb Callback, filename string) error {
		if cb.OnCreate == nil {
			return nil
		}
		return cb.OnCreate(filename)
	})

	dispatchFunc(files.updated, callbacks.match, func(cb Callback, filename string) error {
		if cb.OnUpdate == nil {
			return nil
		}
		return cb.OnUpdate(filename)
	})

	dispatchFunc(files.deleted, callbacks.match, func(cb Callback, filename string) error {
		if cb.OnDelete == nil {
			return nil
		}
		return cb.OnDelete(filename)
	})
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
			dispatch(events, callbacks)

		default:
			paths := slices.Collect(maps.Keys(files))
			newFiles, err := checkFiles(paths)
			if err != nil {
				return err
			}
			events := diff(files, newFiles)
			dispatch(events, callbacks)
			// TODO: config
			time.Sleep(250 * time.Millisecond)
		}
	}

	// err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
	// 	if err != nil {
	// 		return err
	// 	}

	// 	if d.IsDir() {
	// 		return nil
	// 	}

	// 	matches := callbacks.match(path)
	// 	if len(matches) == 0 {
	// 		return nil
	// 	}

	// 	info, err := d.Info()
	// 	if err != nil {
	// 		return err
	// 	}

	// 	modTime := info.ModTime()
	// 	lastModTime, exists := fileInfo[path]
	// 	switch {
	// 	case exists && modTime.After(lastModTime):
	// 		for _, cb := range matches {
	// 			cb.OnUpdate(path)
	// 		}

	// 	case !exists:
	// 		for _, cb := range matches {
	// 			cb.OnCreate(path)
	// 		}
	// 	}
	// 	fileInfo[path] = modTime

	// 	return nil
	// })
}
