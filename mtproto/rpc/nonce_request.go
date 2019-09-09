package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/9seconds/mtg/mtproto"
)

type NonceRequest struct {
	KeySelector []byte
	CryptoTS    []byte
	Nonce       []byte
}

// Bytes returns serialized nonce request.
func (r *NonceRequest) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(mtproto.TagNonce)       // nolint: gosec
	buf.Write(r.KeySelector)          // nolint: gosec
	buf.Write(mtproto.NonceCryptoAES) // nolint: gosec
	buf.Write(r.CryptoTS)             // nolint: gosec
	buf.Write(r.Nonce)                // nolint: gosec

	return buf.Bytes()
}

// NewNonceRequest builds new none request based on proxy secret.
func NewNonceRequest(proxySecret []byte) (*NonceRequest, error) {
	nonce := make([]byte, 16)
	keySelector := make([]byte, 4)
	cryptoTS := make([]byte, 4)

	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("cannot generate nonce: %w", err)
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
