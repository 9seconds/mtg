package rpc

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"time"

	"github.com/juju/errors"
)

const (
	rpcNonceLength            = 16
	rpcNonceKeySelectorLength = 4
	rpcNonceCryptoTSLength    = 4
	rpcNonceTagLength         = 4
	rpcNonceCryptoAESLength   = 4

	rpcNonceRequestLength = rpcNonceTagLength + rpcNonceKeySelectorLength +
		rpcNonceCryptoAESLength + rpcNonceCryptoTSLength + rpcNonceLength
)

var (
	rpcNonceTag          = [rpcNonceTagLength]byte{0xaa, 0x87, 0xcb, 0x7a}
	rpcNonceCryptoAESTag = [rpcNonceCryptoAESLength]byte{0x01, 0x00, 0x00, 0x00}
)

type RPCNonceRequest struct {
	KeySelector [rpcNonceKeySelectorLength]byte
	CryptoTS    [rpcNonceCryptoTSLength]byte
	Nonce       [rpcNonceLength]byte
}

func (r *RPCNonceRequest) Bytes() *bytes.Buffer {
	buf := &bytes.Buffer{}
	buf.Grow(rpcNonceRequestLength)

	buf.Write(rpcNonceTag[:])
	buf.Write(r.KeySelector[:])
	buf.Write(rpcNonceCryptoAESTag[:])
	buf.Write(r.CryptoTS[:])
	buf.Write(r.Nonce[:])

	return buf
}

func NewRPCNonceRequest(proxySecret []byte) (*RPCNonceRequest, error) {
	var nonce [rpcNonceLength]byte
	var keySelector [rpcNonceKeySelectorLength]byte
	var cryptoTS [rpcNonceCryptoTSLength]byte

	if _, err := rand.Read(nonce[:]); err != nil {
		return nil, errors.Annotate(err, "Cannot generate nonce")
	}
	copy(keySelector[:], proxySecret)

	timestamp := time.Now().Truncate(time.Second).Unix() % 4294967296 // 256 ^ 4 - do not know how to name
	binary.LittleEndian.PutUint32(cryptoTS[:], uint32(timestamp))

	return &RPCNonceRequest{
		KeySelector: keySelector,
		CryptoTS:    cryptoTS,
		Nonce:       nonce,
	}, nil
}
