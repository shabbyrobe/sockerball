package main

import (
	"context"
	"encoding/binary"
	"os"

	"github.com/shabbyrobe/cmdy"
)

var encoding = binary.BigEndian

func main() {
	if err := run(); err != nil {
		cmdy.Fatal(err)
	}
}

func run() error {
	// defer profile.Start().Stop()

	bld := func() cmdy.Command {
		return cmdy.NewGroup("socketjunk", cmdy.Builders{
			"client": func() cmdy.Command { return &clientCommand{} },
			"server": func() cmdy.Command { return &serverCommand{} },
		})
	}

	return cmdy.Run(context.Background(), os.Args[1:], bld)
}
