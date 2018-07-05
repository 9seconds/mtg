package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

type RPCHandshakeResponse struct {
	Type      []byte
	Flags     []byte
	SenderPID []byte
	PeerPID   []byte
}

func (r *RPCHandshakeResponse) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(r.Type[:])
	buf.Write(r.Flags[:])
	buf.Write(r.SenderPID[:])
	buf.Write(r.PeerPID[:])

	return buf.Bytes()
}

func (r *RPCHandshakeResponse) Valid(req *RPCHandshakeRequest) error {
	if !bytes.Equal(r.Type, RPCTagHandshake) {
		return errors.New("Unexpected handshake tag")
	}
	if !bytes.Equal(r.PeerPID, RPCHandshakeSenderPID) {
		return errors.New("Incorrect sender PID")
	}

	return nil
}

func NewRPCHandshakeResponse(data []byte) (*RPCHandshakeResponse, error) {
	if len(data) != 32 {
		return nil, errors.New("Incorrect handshake response length")
	}

	return &RPCHandshakeResponse{
		Type:      data[:4],
		Flags:     data[4:8],
		SenderPID: data[8:20],
		PeerPID:   data[20:],
	}, nil
}
