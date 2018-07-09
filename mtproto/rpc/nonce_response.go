package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

type NonceResponse struct {
	NonceRequest

	Type   []byte
	Crypto []byte
}

func (r *NonceResponse) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(r.Type)
	buf.Write(r.KeySelector)
	buf.Write(r.Crypto)
	buf.Write(r.CryptoTS)
	buf.Write(r.Nonce)

	return buf.Bytes()
}

func (r *NonceResponse) Valid(req *NonceRequest) error {
	if !bytes.Equal(r.Type, TagNonce) {
		return errors.New("Unexpected RPC type")
	}
	if !bytes.Equal(r.Crypto, NonceCryptoAES) {
		return errors.New("Unexpected crypto type")
	}
	if !bytes.Equal(r.KeySelector, req.KeySelector) {
		return errors.New("Unexpected key selector")
	}

	return nil
}

func NewNonceResponse(data []byte) (*NonceResponse, error) {
	if len(data) != 32 {
		return nil, errors.New("Unexpected message length")
	}

	return &NonceResponse{
		NonceRequest: NonceRequest{
			KeySelector: data[4:8],
			CryptoTS:    data[12:16],
			Nonce:       data[16:],
		},
		Type:   data[:4],
		Crypto: data[8:12],
	}, nil
}
