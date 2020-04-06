package main

import (
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/example"
	"github.com/shabbyrobe/sockerball/example/exampleproto"
)

type clientCommand struct {
	host    string
	spammer exampleproto.Spammer
}

func (cl *clientCommand) Help() cmdy.Help { return cmdy.Synopsis("client") }

func (cl *clientCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	cl.spammer.Flags(flags)
	args.StringOptional(&cl.host, "host", "localhost:9631", "host")
}

func (cl *clientCommand) Run(ctx cmdy.Context) error {
	example.DebugServer(":14440")

	dialer := cl.spammer.Dialer(exampleproto.Negotiator(1, 3))

	clientCb := func(handler sockerball.Handler, opts ...sockerball.ClientOption) (sockerball.Client, error) {
		client, err := dialer.DialStream(ctx, "tcp", cl.host, handler, opts...)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	return cl.spammer.Spam(ctx, nil, clientCb)
}
