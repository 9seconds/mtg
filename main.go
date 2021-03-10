package main

import (
	"math/rand"
	"time"

	"github.com/alecthomas/kong"
)

var version = "dev" // has to be set by ldflags

type CLI struct {
	GenerateSecret struct { // nolint: govet
		HostName string `arg optional help:"Hostname to use for domain fronting. Default is '${domain_front}'." name:"hostname" default:"${domain_front}"` // nolint: lll, govet
		Hex      bool   `help:"Print secret in hex encoding."`
	} `cmd help:"Generate new proxy secret."`
	Access struct { // nolint: govet
		ConfigPath string `arg required type:"existingfile" help:"Path to the configuration file." name:"config-path"` // nolint: lll, govet
		Hex        bool   `help:"Print secret in hex encoding."`
	} `cmd help:"Print access information."`
	Run struct { // nolint: govet
		ConfigPath string `arg required type:"existingfile" help:"Path to the configuration file." name:"config-path"` // nolint: lll, govet
	} `cmd help:"Run proxy."`
	Version kong.VersionFlag `help:"Print version."`
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cli := &CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"domain_front": "amazonaws.com",
		"config_path":  "/etc/mtg.toml",
		"version":      version,
	})

	switch ctx.Command() {
	case "generate-secret":
		runGenerateSecret(cli)
	case "access <config-path>":
		runAccess(cli)
	case "run <config-path>":
		panic("not implemented yet")
	default:
		panic(ctx.Command())
	}
}
