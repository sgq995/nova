package codegen

import "strconv"

type MessageType int

const (
	BulkType MessageType = iota

	CreateFileType
	UpdateFileType
	DeleteFileType

	CreateRouteType
	DeleteRouteType
)

func (t MessageType) Int() int {
	return int(t)
}

func (t MessageType) Itoa() string {
	return strconv.Itoa(t.Int())
}

func (t MessageType) String() string {
	switch t {
	case BulkType:
		return "BulkType"

	case CreateFileType:
		return "CreateFileType"

	case UpdateFileType:
		return "UpdateFileType"

	case DeleteFileType:
		return "DeleteFileType"

	case CreateRouteType:
		return "CreateRouteType"

	case DeleteRouteType:
		return "DeleteRouteType"

	default:
		return ""
	}
}

type Message struct {
	Type    MessageType    `json:"type"`
	Payload map[string]any `json:"payload"`
}

func BulkMessage(messages ...*Message) *Message {
	return &Message{
		Type: BulkType,
		Payload: map[string]any{
			"messages": messages,
		},
	}
}

func CreateFileMessage(filename string, contents []byte) *Message {
	return &Message{
		Type: CreateFileType,
		Payload: map[string]any{
			"filename": filename,
			"contents": contents,
		},
	}
}

func UpdateFileMessage(filename string, contents []byte) *Message {
	return &Message{
		Type: UpdateFileType,
		Payload: map[string]any{
			"filename": filename,
			"contents": contents,
		},
	}
}

func DeleteFileMessage(filename string) *Message {
	return &Message{
		Type: DeleteFileType,
		Payload: map[string]any{
			"filename": filename,
		},
	}
}

func CreateRouteMessage(pattern string) *Message {
	return &Message{
		Type: CreateRouteType,
		Payload: map[string]any{
			"pattern": pattern,
		},
	}
}

func DeleteRouteMessage(pattern string) *Message {
	return &Message{
		Type: DeleteRouteType,
		Payload: map[string]any{
			"pattern": pattern,
		},
	}
}
