package watcher

import (
	"path/filepath"
	"strings"

	"github.com/sgq995/nova/internal/utils"
)

type CallbackFunc func(string) error

type Callback struct {
	OnCreate CallbackFunc
	OnUpdate CallbackFunc
	OnDelete CallbackFunc
}

type eventCallbackMap map[string]CallbackFunc

func (ecm eventCallbackMap) match(filename string) []CallbackFunc {
	name := filepath.Base(filename)
	matches := []CallbackFunc{}
	for matcher, cb := range ecm {
		patterns := strings.Split(matcher, ",")
		for _, pattern := range patterns {
			if matched := utils.Must(filepath.Match(pattern, name)); matched {
				matches = append(matches, cb)
			}
		}
	}
	return matches
}

type CallbackMap map[string]Callback

func (cm CallbackMap) splitEventCallbacks() (eventCallbackMap, eventCallbackMap, eventCallbackMap) {
	created := eventCallbackMap{}
	updated := eventCallbackMap{}
	deleted := eventCallbackMap{}

	for pattern, cb := range cm {
		if cb.OnCreate != nil {
			created[pattern] = cb.OnCreate
		}

		if cb.OnUpdate != nil {
			updated[pattern] = cb.OnUpdate
		}

		if cb.OnDelete != nil {
			deleted[pattern] = cb.OnDelete
		}
	}

	return created, updated, deleted
}
