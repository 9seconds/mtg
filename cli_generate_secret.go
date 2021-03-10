package main

import (
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
)

type cliCommandGenerateSecret struct {
	HostName string `arg optional help:"Hostname to use for domain fronting. Default is '${domain_front}'." name:"hostname" default:"${domain_front}"` // nolint: lll, govet
	Hex      bool   `help:"Print secret in hex encoding."`
}

func (c *cliCommandGenerateSecret) Run(cli *CLI) error { // nolint: unparam
	secret := mtglib.GenerateSecret(cli.GenerateSecret.HostName)

	if cli.GenerateSecret.Hex {
		fmt.Println(secret.Hex()) // nolint: forbidigo
	} else {
		fmt.Println(secret.Base64()) // nolint: forbidigo
	}

	return nil
}
