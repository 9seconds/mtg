package conntypes

import (
	"crypto/rand"
	"encoding/hex"
)

const ConnIDLength = 8

type ConnID [ConnIDLength]byte

func (c ConnID) String() string {
	return hex.EncodeToString(c[:])
}

func NewConnID() ConnID {
	var id ConnID

	if _, err := rand.Read(id[:]); err != nil {
		panic(err)
	}

	return id
}
