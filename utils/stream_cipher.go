package utils

import (
	"crypto/aes"
	"crypto/cipher"
)

func MakeStreamCipher(key, iv []byte) cipher.Stream {
	block, _ := aes.NewCipher(key) // nolint: gosec
	return cipher.NewCTR(block, iv)
}
