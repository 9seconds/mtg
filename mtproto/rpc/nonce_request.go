package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/juju/errors"
)

type NonceRequest struct {
	KeySelector []byte
	CryptoTS    []byte
	Nonce       []byte
}

func (r *NonceRequest) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(TagNonce)
	buf.Write(r.KeySelector)
	buf.Write(NonceCryptoAES)
	buf.Write(r.CryptoTS)
	buf.Write(r.Nonce)

	return buf.Bytes()
}

func NewNonceRequest(proxySecret []byte) (*NonceRequest, error) {
	nonce := make([]byte, 16)
	keySelector := make([]byte, 4)
	cryptoTS := make([]byte, 4)

	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.Annotate(err, "Cannot generate nonce")
	}
	copy(keySelector, proxySecret)

	timestamp := time.Now().Truncate(time.Second).Unix() % 4294967296 // 256 ^ 4 - do not know how to name
	binary.LittleEndian.PutUint32(cryptoTS, uint32(timestamp))

	return &NonceRequest{
		KeySelector: keySelector,
		CryptoTS:    cryptoTS,
		Nonce:       nonce,
	}, nil
}