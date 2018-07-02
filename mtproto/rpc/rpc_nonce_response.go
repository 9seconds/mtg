package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

const rpcNonceResponseLength = rpcNonceRequestLength

type RPCNonceResponse struct {
	RPCNonceRequest

	RPCType [rpcNonceTagLength]byte
	Crypto  [rpcNonceCryptoAESLength]byte
}

func (r *RPCNonceResponse) Bytes() []byte {
	buf := &bytes.Buffer{}
	buf.Grow(rpcNonceResponseLength)

	buf.Write(r.RPCType[:])
	buf.Write(r.KeySelector[:])
	buf.Write(r.Crypto[:])
	buf.Write(r.CryptoTS[:])
	buf.Write(r.Nonce[:])

	return buf.Bytes()
}

func (r *RPCNonceResponse) Valid(req *RPCNonceRequest) error {
	if r.RPCType != rpcNonceTag {
		return errors.New("Unexpected RPC type")
	}
	if r.Crypto != rpcNonceCryptoAESTag {
		return errors.New("Unexpected crypto type")
	}
	if r.KeySelector != req.KeySelector {
		return errors.New("Unexpected key selector")
	}

	return nil
}

func NewRPCNonceResponse(data []byte) (*RPCNonceResponse, error) {
	if len(data) != rpcNonceResponseLength {
		return nil, errors.New("Unexpected message length")
	}

	resp := RPCNonceResponse{}
	copy(resp.RPCType[:], data[:4])
	copy(resp.KeySelector[:], data[4:8])
	copy(resp.Crypto[:], data[8:12])
	copy(resp.CryptoTS[:], data[12:16])
	copy(resp.Nonce[:], data[16:])

	return &resp, nil
}
