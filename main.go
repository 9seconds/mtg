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
	// this runs profiling server. To enable it, build with prof tag
	//   $ go build -tags prof
	//
	// Then you can pass a port using MTG_PROF_PORT environment variable.
	// Default is 6000
	//   $ MTG_PROF_PORT=6000 mtg run config.toml
	//
	// It will run a webserver with profiling data on
	// localhost:${MTG_PROF_PORT:-6000}.
	//
	// To collect PGO do following:
	//   $ curl -o default.pgo 'http://localhost:6000/debug/pprof/profile?seconds=300'
	//
	// See also https://pkg.go.dev/net/http/pprof
	//          https://go.dev/blog/pprof
	runProfile()

	cli := &cli.CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"version": getVersion(),
	})

	ctx.FatalIfErrorf(ctx.Run(cli, version))
}
