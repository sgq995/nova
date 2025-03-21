package watcher

type Event int

const (
	CreateEvent Event = iota
	UpdateEvent
	DeleteEvent
)

func (e Event) String() string {
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

type CallbackFunc func(event Event, files []string) error

type CallbackMap map[string]CallbackFunc
