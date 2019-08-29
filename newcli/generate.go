package newcli

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/9seconds/mtg/newconfig"
)

func Generate(secretType string) {
	data := make([]byte, newconfig.SimpleSecretLength)
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
