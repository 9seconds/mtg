// mtg is just a command-line application that starts a proxy.
//
// Application logic is how to read a config and configure mtglib.Proxy.
// So, probably you need to read the documentation for mtglib package
// first.
//
// mtglib is a core of the application. The rest of the packages provide
// some default implementations for the interfaces, defined in mtglib.
package main

import (
	"github.com/9seconds/mtg/v2/internal/cli"
	"github.com/alecthomas/kong"
)

func main() {
	cli := &cli.CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"version": getVersion(),
	})

	ctx.FatalIfErrorf(ctx.Run(cli, version))
}
