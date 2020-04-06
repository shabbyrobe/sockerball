package main

import (
	"fmt"
	"time"

	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/example"
	"github.com/shabbyrobe/sockerball/example/exampleproto"
)

type serverCommand struct {
	host string
}

func (sc *serverCommand) Help() cmdy.Help { return cmdy.Synopsis("server") }

func (sc *serverCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.StringVar(&sc.host, "host", ":9631", "host")
}

func (sc *serverCommand) Run(ctx cmdy.Context) error {
	example.DebugServer(":14441")

	handler := &exampleproto.ServerHandler{}

	var config = sockerball.DefaultServerConfig()
	config.Conn.ResponseTimeout = 20 * time.Second
	config.Conn.ReadTimeout = 20 * time.Second
	config.Conn.WriteTimeout = 20 * time.Second

	ln, err := sockerball.ListenStream("tcp", sc.host)
	if err != nil {
		return err
	}

	srv := sockerball.NewServer(config, exampleproto.Negotiator(0, 0), handler,
		sockerball.ServerConnect(func(srv *sockerball.Server, id sockerball.ConnID) {
			fmt.Println("connect", id)
		}),
		sockerball.ServerDisconnect(func(srv *sockerball.Server, id sockerball.ConnID, err error) {
			fmt.Println("disconnect", id, err)
		}),
	)

	fmt.Printf("listening on %s\n", sc.host)

	return srv.Serve(ln)
}
