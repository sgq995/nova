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

type CallbackMap map[string]Callback

// TODO: split callbacks by groups: create, update, delete and filter out nil callbacks
func (cm CallbackMap) match(filename string) []Callback {
	name := filepath.Base(filename)
	matches := []Callback{}
	for matcher := range cm {
		patterns := strings.Split(matcher, ",")
		for _, pattern := range patterns {
			if matched := utils.Must(filepath.Match(pattern, name)); matched {
				matches = append(matches, cm[matcher])
			}
		}
	}
	return matches
}
