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
	"fmt"
	"math/rand"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/9seconds/mtg/v2/internal/cli"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/alecthomas/kong"
)

var version = "dev" // has to be set by ldflags

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	if err := utils.SetLimits(); err != nil {
		panic(err)
	}

	if buildInfo, ok := debug.ReadBuildInfo(); ok {
		vcsCommit := "<no-commit>"
		vcsDate := time.Now()
		vcsDirty := ""

		for _, setting := range buildInfo.Settings {
			switch setting.Key {
			case "vcs.time":
				vcsDate, _ = time.Parse(time.RFC3339, setting.Value)
			case "vcs.revision":
				vcsCommit = setting.Value
			case "vcs.modified":
				if isDirty, _ := strconv.ParseBool(setting.Value); isDirty {
					vcsDirty = " [dirty]"
				}
			}
		}

		version = fmt.Sprintf("%s (%s: %s on %s%s)",
			version,
			buildInfo.GoVersion,
			vcsDate.Format(time.RFC3339),
			vcsCommit,
			vcsDirty)
	}

	cli := &cli.CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"version": version,
	})

	ctx.FatalIfErrorf(ctx.Run(cli, version))
}
