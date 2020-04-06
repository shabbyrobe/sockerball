package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/shabbyrobe/cmdy"
	"github.com/shabbyrobe/cmdy/arg"
	"github.com/shabbyrobe/sockerball"
	"github.com/shabbyrobe/sockerball/example/exampleproto"
	"github.com/shabbyrobe/sockerball/packetsrv"
)

type clientCommand struct {
	host string
}

func (cl *clientCommand) Help() cmdy.Help { return cmdy.Synopsis("pktclient") }

func (cl *clientCommand) Configure(flags *cmdy.FlagSet, args *arg.ArgSet) {
	args.StringOptional(&cl.host, "host", "localhost:9633", "host")
}

func (cl *clientCommand) Run(ctx cmdy.Context) error {
	socketDialer := sockerball.DefaultDialer(exampleproto.Negotiator(0, 0))

	dialer := net.Dialer{}
	handler := &exampleproto.ServerHandler{}

	in, err := ioutil.ReadFile("/Users/bl/Downloads/The Rust Programming Language.htm")
	if err != nil {
		return err
	}
	in = in[:50]
	_ = in

	s := time.Now()

	iter := 200
	threads := 1000
	var wg sync.WaitGroup
	wg.Add(threads)

	for thread := 0; thread < threads; thread++ {
		go func(thread int) {
			defer wg.Done()

			conn, err := dialer.DialContext(ctx, "udp", cl.host)
			if err != nil {
				panic(err)
			}
			client, err := socketDialer.Client(ctx, packetsrv.ClientCommunicator(conn), handler)
			if err != nil {
				panic(err)
			}

			rq := &exampleproto.TestRequest{
				Foo: fmt.Sprintf("%d", thread),
			}
			for i := 0; i < iter; i++ {
				rsp, err := (client.Request(ctx, rq))
				if err != nil {
					panic(err)
				}
				_ = rsp
				// fmt.Printf("%#v\n", rsp)
			}
		}(thread)
	}

	wg.Wait()
	spew.Dump(time.Since(s))

	return nil
}
