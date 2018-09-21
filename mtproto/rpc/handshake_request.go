package rpc

import "bytes"

// HandshakeRequest is the data type which is responsible for
// constructing of correct handshake request.
type HandshakeRequest struct {
}

// Bytes returns serialized handshake request.
func (r *HandshakeRequest) Bytes() []byte {
	buf := &bytes.Buffer{}
	buf.Grow(len(TagHandshake) + len(HandshakeFlags) + len(HandshakeSenderPID) + len(HandshakePeerPID))

	buf.Write(TagHandshake)       // nolint: gosec
	buf.Write(HandshakeFlags)     // nolint: gosec
	buf.Write(HandshakeSenderPID) // nolint: gosec
	buf.Write(HandshakePeerPID)   // nolint: gosec

	return buf.Bytes()
}

// NewHandshakeRequest creates new HandshakeRequest instance.
func NewHandshakeRequest() *HandshakeRequest {
	return &HandshakeRequest{}
}
