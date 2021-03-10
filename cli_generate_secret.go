package main

import (
	"fmt"

	"github.com/9seconds/mtg/v2/mtglib"
)

func runGenerateSecret(cli *CLI) {
	secret := mtglib.GenerateSecret(cli.GenerateSecret.HostName)

	if cli.GenerateSecret.Hex {
		fmt.Println(secret.Hex()) // nolint: forbidigo
	} else {
		fmt.Println(secret.Base64()) // nolint: forbidigo
	}
}
