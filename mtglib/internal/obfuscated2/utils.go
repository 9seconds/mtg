package obfuscated2

import (
	"crypto/aes"
	"crypto/cipher"
)

func makeAesCtr(key, iv []byte) cipher.Stream {
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err)
	}

	return cipher.NewCTR(block, iv)
}
