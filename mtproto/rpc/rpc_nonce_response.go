package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

type RPCNonceResponse struct {
	RPCNonceRequest

	RPCType []byte
	Crypto  []byte
}

func (r *RPCNonceResponse) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(r.RPCType)
	buf.Write(r.KeySelector)
	buf.Write(r.Crypto)
	buf.Write(r.CryptoTS)
	buf.Write(r.Nonce)

	return buf.Bytes()
}

func (r *RPCNonceResponse) Valid(req *RPCNonceRequest) error {
	if !bytes.Equal(r.RPCType, RPCTagNonce) {
		return errors.New("Unexpected RPC type")
	}
	if !bytes.Equal(r.Crypto, RPCNonceCryptoAES) {
		return errors.New("Unexpected crypto type")
	}
	if !bytes.Equal(r.KeySelector, req.KeySelector) {
		return errors.New("Unexpected key selector")
	}

	return nil
}

func NewRPCNonceResponse(data []byte) (*RPCNonceResponse, error) {
	if len(data) != 32 {
		return nil, errors.New("Unexpected message length")
	}

	return &RPCNonceResponse{
		RPCNonceRequest: RPCNonceRequest{
			KeySelector: data[4:8],
			CryptoTS:    data[12:16],
			Nonce:       data[16:],
		},
		RPCType: data[:4],
		Crypto:  data[8:12],
	}, nil
}
