package rpc

import (
	"bytes"
	"errors"
	"fmt"
)

type HandshakeResponse struct {
	Type      []byte
	Flags     []byte
	SenderPID []byte
	PeerPID   []byte
}

// Bytes returns a serialized handshake response.
func (r *HandshakeResponse) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(r.Type)      // nolint: gosec
	buf.Write(r.Flags)     // nolint: gosec
	buf.Write(r.SenderPID) // nolint: gosec
	buf.Write(r.PeerPID)   // nolint: gosec

	return buf.Bytes()
}

// Valid checks that handshake response compliments request.
func (r *HandshakeResponse) Valid() error {
	if !bytes.Equal(r.Type, TagHandshake) {
		return errors.New("unexpected handshake tag")
	}

	if !bytes.Equal(r.PeerPID, HandshakeSenderPID) {
		return errors.New("incorrect sender PID")
	}

	return nil
}

// NewHandshakeResponse constructs new handshake response from the given
// data.
func NewHandshakeResponse(data []byte) (*HandshakeResponse, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("incorrect handshake response length %d", len(data))
	}

	return &HandshakeResponse{
		Type:      data[:4],
		Flags:     data[4:8],
		SenderPID: data[8:20],
		PeerPID:   data[20:],
	}, nil
}
