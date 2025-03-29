package watcher

import (
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/must"
)

func dispatchEvent(callbacks CallbackMap, event Event, files []string) {
	for matcher, cb := range callbacks {
		target := make([]string, 0)
		patterns := strings.Split(matcher, ",")
		for _, pattern := range patterns {
			for _, filename := range files {
				name := filepath.Base(filename)
				if must.Must(filepath.Match(pattern, name)) {
					target = append(target, filename)
				}
			}
		}
		if len(target) > 0 {
			go func() {
				err := cb(event, target)
				if err != nil {
					logger.Errorf("%+v", err)
				}
			}()
		}
	}
}

func dispatch(files *fileEvents, callbacks CallbackMap) {
	dispatchEvent(callbacks, CreateEvent, files.created)
	dispatchEvent(callbacks, UpdateEvent, files.updated)
	dispatchEvent(callbacks, DeleteEvent, files.deleted)
}
