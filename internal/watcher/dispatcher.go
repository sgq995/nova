package watcher

import (
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/logger"
	"github.com/sgq995/nova/internal/utils"
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

func dispatchEvent(callbacks WatchCallbackMap, event WatchEvent, files []string) {
	for matcher, cb := range callbacks {
		target := make([]string, 0)
		patterns := strings.Split(matcher, ",")
		for _, pattern := range patterns {
			for _, filename := range files {
				name := filepath.Base(filename)
				if utils.Must(filepath.Match(pattern, name)) {
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

func dispatch(files *fileEvents, callbacks WatchCallbackMap) {
	dispatchEvent(callbacks, CreateEvent, files.created)
	dispatchEvent(callbacks, UpdateEvent, files.updated)
	dispatchEvent(callbacks, DeleteEvent, files.deleted)
}
