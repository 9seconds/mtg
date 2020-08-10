package utils

import (
	"crypto/aes"
	"crypto/cipher"
)

func MakeStreamCipher(key, iv []byte) cipher.Stream {
	block, _ := aes.NewCipher(key)

	return cipher.NewCTR(block, iv)
}
