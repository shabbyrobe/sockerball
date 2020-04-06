package exampleproto

import (
	"encoding/json"
	"fmt"

	"github.com/shabbyrobe/sockerball"
)

type JSONEnvelope struct {
	ID      sockerball.MessageID
	ReplyTo sockerball.MessageID
	Kind    int
	Message json.RawMessage
}

type Mapper struct{}

func (m Mapper) Message(kind int) (sockerball.Message, error) {
	switch kind {
	case 1:
		return &TestRequest{}, nil
	case 2:
		return &TestResponse{}, nil
	case 3:
		return &TestCommand{}, nil
	case 4:
		return &OK{}, nil
	default:
		return nil, fmt.Errorf("unknown kind %d", kind)
	}
}

func (m Mapper) MessageKind(msg sockerball.Message) (int, error) {
	switch msg.(type) {
	case *TestRequest:
		return 1, nil
	case *TestResponse:
		return 2, nil
	case *TestCommand:
		return 3, nil
	case *OK:
		return 4, nil
	default:
		return 0, fmt.Errorf("unknown msg %T", msg)
	}
}
