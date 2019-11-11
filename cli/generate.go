package cli

import (
	"crypto/rand"
	"encoding/hex"

	"mtg/config"
)

func Generate(secretType, hostname string) {
	data := make([]byte, config.SimpleSecretLength)
	if _, err := rand.Read(data); err != nil {
		panic(err)
	}

	secret := hex.EncodeToString(data)

	switch secretType {
	case "simple":
		PrintStdout(secret)
	case "secured":
		PrintStdout("dd" + secret)
	default:
		PrintStdout("ee" + secret + hex.EncodeToString([]byte(hostname)))
	}
}
