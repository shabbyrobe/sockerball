package main

import (
	"fmt"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/example/exampleproto"
	"github.com/shabbyrobe/sockerball/packetsrv"
)

type serverCommand struct {
	host string
}

func (sc *serverCommand) Help() cmdy.Help { return cmdy.Synopsis("pktserver") }

func (sc *serverCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.StringVar(&sc.host, "host", ":9633", "host")
}

func (sc *serverCommand) Run(ctx cmdy.Context) error {
	handler := &exampleproto.ServerHandler{}

	ln, err := packetsrv.Listen("udp", sc.host)
	if err != nil {
		return err
	}

	fmt.Printf("listening on %s\n", sc.host)
	srv := sockerball.NewServer(nil, exampleproto.Negotiator(0, 0), handler)
	return srv.Serve(ln)
}
