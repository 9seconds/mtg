package rpc

import (
	"bytes"

	"github.com/juju/errors"
)

type HandshakeResponse struct {
	Type      []byte
	Flags     []byte
	SenderPID []byte
	PeerPID   []byte
}

func (r *HandshakeResponse) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(r.Type[:])
	buf.Write(r.Flags[:])
	buf.Write(r.SenderPID[:])
	buf.Write(r.PeerPID[:])

	return buf.Bytes()
}

func (r *HandshakeResponse) Valid(req *HandshakeRequest) error {
	if !bytes.Equal(r.Type, TagHandshake) {
		return errors.New("Unexpected handshake tag")
	}
	if !bytes.Equal(r.PeerPID, HandshakeSenderPID) {
		return errors.New("Incorrect sender PID")
	}

	return nil
}

func NewHandshakeResponse(data []byte) (*HandshakeResponse, error) {
	if len(data) != 32 {
		return nil, errors.New("Incorrect handshake response length")
	}

	return &HandshakeResponse{
		Type:      data[:4],
		Flags:     data[4:8],
		SenderPID: data[8:20],
		PeerPID:   data[20:],
	}, nil
}
