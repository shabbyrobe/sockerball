package main

import (
	"time"

	"github.com/gorilla/websocket"
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/example/exampleproto"
	"github.com/shabbyrobe/sockerball/wsocketsrv"
)

type clientCommand struct {
	url     string
	spammer exampleproto.Spammer
}

func (cl *clientCommand) Help() cmdy.Help { return cmdy.Synopsis("wsclient") }

func (cl *clientCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	cl.spammer.Flags(flags)
	args.StringOptional(&cl.url, "url", "ws://localhost:9632/", "host")
}

func (cl *clientCommand) Run(ctx cmdy.Context) error {
	// defer profile.Start().Stop()

	dialer := cl.spammer.Dialer(exampleproto.Negotiator(0, 0))

	clientCb := func(handler sockerball.Handler, opts ...sockerball.ClientOption) (sockerball.Client, error) {
		wsDialer := websocket.Dialer{
			HandshakeTimeout: 5 * time.Second,
		}
		sock, _, err := wsDialer.Dial(cl.url, nil)
		if err != nil {
			return nil, err
		}
		client, err := dialer.Client(ctx, wsocketsrv.NewCommunicator(sock), handler, opts...)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	return cl.spammer.Spam(ctx, nil, clientCb)
}
