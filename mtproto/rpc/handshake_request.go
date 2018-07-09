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

	buf.Write(TagHandshake)
	buf.Write(HandshakeFlags)
	buf.Write(HandshakeSenderPID)
	buf.Write(HandshakePeerPID)

	return buf.Bytes()
}

// NewHandshakeRequest creates new HandshakeRequest instance.
func NewHandshakeRequest() *HandshakeRequest {
	return &HandshakeRequest{}
}
