package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/juju/errors"
)

type RPCNonceRequest struct {
	KeySelector []byte
	CryptoTS    []byte
	Nonce       []byte
}

func (r *RPCNonceRequest) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(RPCTagNonce)
	buf.Write(r.KeySelector)
	buf.Write(RPCNonceCryptoAES)
	buf.Write(r.CryptoTS)
	buf.Write(r.Nonce)

	return buf.Bytes()
}

func NewRPCNonceRequest(proxySecret []byte) (*RPCNonceRequest, error) {
	nonce := make([]byte, 16)
	keySelector := make([]byte, 4)
	cryptoTS := make([]byte, 4)

	if _, err := rand.Read(nonce); err != nil {
		return nil, errors.Annotate(err, "Cannot generate nonce")
	}
	copy(keySelector, proxySecret)

	timestamp := time.Now().Truncate(time.Second).Unix() % 4294967296 // 256 ^ 4 - do not know how to name
	binary.LittleEndian.PutUint32(cryptoTS, uint32(timestamp))

	return &RPCNonceRequest{
		KeySelector: keySelector,
		CryptoTS:    cryptoTS,
		Nonce:       nonce,
	}, nil
}
