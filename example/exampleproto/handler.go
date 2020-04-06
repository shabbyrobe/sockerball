package exampleproto

import (
	"context"

	"github.com/shabbyrobe/sockerball"
)

type ServerHandler struct{}

func (h ServerHandler) HandleRequest(ctx context.Context, in sockerball.IncomingRequest) (rs sockerball.Message, rerr error) {
	switch msg := in.Message.(type) {
	case *TestRequest:
		return &TestResponse{Bar: msg.Foo + ": yep!"}, nil
	default:
		return &OK{}, nil
	}
}
