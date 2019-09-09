package rpc

import (
	"bytes"
	"errors"
	"fmt"
)

type NonceResponse struct {
	NonceRequest

	Type   []byte
	Crypto []byte
}

// Bytes returns serialized form of the nonce response.
func (r *NonceResponse) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(r.Type)        // nolint: gosec
	buf.Write(r.KeySelector) // nolint: gosec
	buf.Write(r.Crypto)      // nolint: gosec
	buf.Write(r.CryptoTS)    // nolint: gosec
	buf.Write(r.Nonce)       // nolint: gosec

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

// NewNonceResponse build new nonce response based on the given data.
func NewNonceResponse(data []byte) (*NonceResponse, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("Unexpected message length %d", len(data))
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
