package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

const rpcHandshakeResponseLength = rpcHandshakeRequestLength

type RPCHandshakeResponse struct {
	Type      [rpcHandshakeTagLength]byte
	Flags     [rpcHandshakeFlagsLength]byte
	SenderPID [rpcHandshakeSenderPIDLength]byte
	PeerPID   [rpcHandshakePeerPIDLength]byte
}

func (r *RPCHandshakeResponse) Bytes() *bytes.Buffer {
	buf := &bytes.Buffer{}
	buf.Grow(rpcHandshakeResponseLength)

	buf.Write(r.Type[:])
	buf.Write(r.Flags[:])
	buf.Write(r.SenderPID[:])
	buf.Write(r.PeerPID[:])

	return buf
}

func (r *RPCHandshakeResponse) Valid(req *RPCHandshakeRequest) error {
	if r.Type != rpcHandshakeTag {
		return errors.New("Unexpected handshake tag")
	}
	if r.PeerPID != rpcHandshakeSenderPID {
		return errors.New("Incorrect sender PID")
	}

	return nil
}

func NewRPCHandshakeResponse(data []byte) (*RPCHandshakeResponse, error) {
	if len(data) != rpcHandshakeResponseLength {
		return nil, errors.New("Incorrect handshake response length")
	}

	resp := RPCHandshakeResponse{}
	copy(resp.Type[:], data[:4])
	copy(resp.Flags[:], data[4:8])
	copy(resp.SenderPID[:], data[8:20])
	copy(resp.PeerPID[:], data[20:])

	return &resp, nil
}
