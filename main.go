package main

import (
	"math/rand"
	"time"

	"github.com/9seconds/mtg/v2/cli"
	"github.com/alecthomas/kong"
)

var version = "dev" // has to be set by ldflags

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cli := &cli.CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"version": version,
	})

	ctx.FatalIfErrorf(ctx.Run(cli, version))
}
