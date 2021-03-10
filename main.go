package main

import (
	"math/rand"
	"time"

	"github.com/alecthomas/kong"
)

var version = "dev" // has to be set by ldflags

type CLI struct {
	GenerateSecret struct {
		HostName string `arg optional help:"Hostname to use for domain fronting. Default is '${domain_front}'." name:"hostname" default:"${domain_front}"`
		Hex      bool   `help:"Print secret in hex encoding."`
	} `cmd help:"Generate new proxy secret."`
	Access struct {
		ConfigPath string `arg required type:"existingfile" help:"Path to the configuration file." name:"config-path"`
	} `cmd help:"Print access information."`
	Run struct {
		ConfigPath string `arg required type:"existingfile" help:"Path to the configuration file." name:"config-path"`
	} `cmd help:"Run proxy."`
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cli := &CLI{}
	ctx := kong.Parse(cli, kong.Vars{
		"domain_front": "amazonaws.com",
		"config_path":  "/etc/mtg.toml",
	})

	switch ctx.Command() {
	case "generate-secret":
		runGenerateSecret(cli)
	case "access":
	case "run":
		panic("not implemented yet")
	}
}
