package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/profile"
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/example/exampleproto"
	"github.com/shabbyrobe/sockerball/wsocketsrv"
)

type serverCommand struct {
	host string
}

func (sc *serverCommand) Help() cmdy.Help { return cmdy.Synopsis("wsserver") }

func (sc *serverCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	flags.StringVar(&sc.host, "host", ":9632", "host")
}

func (sc *serverCommand) Run(ctx cmdy.Context) error {
	defer profile.Start().Stop()

	ln := wsocketsrv.NewListener(websocket.Upgrader{})

	errc := make(chan error, 2)
	websrv := &http.Server{
		Addr:    sc.host,
		Handler: ln,
	}

	handler := &exampleproto.ServerHandler{}

	srv := sockerball.NewServer(nil, exampleproto.Negotiator(0, 0), handler,
		sockerball.ServerConnect(func(srv *sockerball.Server, id sockerball.ConnID) {
			fmt.Println("connect", id)
		}),
		sockerball.ServerDisconnect(func(srv *sockerball.Server, id sockerball.ConnID, err error) {
			fmt.Println("disconnect", id, err)
		}),
	)

	fmt.Printf("listening on %s\n", sc.host)

	go func() {
		errc <- websrv.ListenAndServe()
	}()
	go func() {
		errc <- srv.Serve(ln)
	}()

	defer func() {
		websrv.Shutdown(context.Background())
		ln.Close()
	}()

	return <-errc
}
