package cli

import (
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
)

type GenerateSecret struct {
	HostName string `kong:"arg,required,help='Hostname to use for domain fronting.',name='hostname'"`
	Hex      bool   `kong:"help='Print secret in hex encoding.',short='x'"`
}

func (g *GenerateSecret) Run(cli *CLI, _ string) error {
	secret := mtglib.GenerateSecret(cli.GenerateSecret.HostName)

	if g.Hex {
		fmt.Println(secret.Hex()) //nolint: forbidigo
	} else {
		fmt.Println(secret.Base64()) //nolint: forbidigo
	}

	return nil
}
