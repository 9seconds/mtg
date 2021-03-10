package main

import (
	"math/rand"
	"time"

	"github.com/alecthomas/kong"
)

var version = "dev" // has to be set by ldflags

type CLI struct {
	GenerateSecret cliCommandGenerateSecret `cmd help:"Generate new proxy secret"` // nolint: govet
	Access         cliCommandAccess         `cmd help:"Print access information."` // nolint: govet
	Version        kong.VersionFlag         `help:"Print version."`
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cli := &CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"domain_front": "amazonaws.com",
		"version":      version,
	})

	ctx.FatalIfErrorf(ctx.Run(cli))
}
