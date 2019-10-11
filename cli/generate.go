package cli

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/9seconds/mtg/config"
)

func Generate(secretType string) {
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
		Fatal("Unknown secret type " + secret)
	}
}
