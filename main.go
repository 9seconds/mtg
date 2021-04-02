package main

import (
	"math/rand"
	"time"

	"github.com/9seconds/mtg/v2/cli"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/alecthomas/kong"
)

var version = "dev" // has to be set by ldflags

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	if err := utils.SetLimits(); err != nil {
		panic(err)
	}

	cli := &cli.CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"version": version,
	})

	ctx.FatalIfErrorf(ctx.Run(cli, version))
}
