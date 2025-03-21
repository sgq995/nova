package watcher

type WatchEvent int

const (
	CreateEvent WatchEvent = iota
	UpdateEvent
	DeleteEvent
)

func (e WatchEvent) String() string {
	switch e {
	case CreateEvent:
		return "CreateEvent"
	case UpdateEvent:
		return "UpdateEvent"
	case DeleteEvent:
		return "DeleteEvent"
	default:
		return ""
	}
}

type WatchCallback func(event WatchEvent, files []string) error

type WatchCallbackMap map[string]WatchCallback
