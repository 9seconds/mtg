package rpc

import "bytes"

type HandshakeRequest struct {
}

func (r *HandshakeRequest) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(TagHandshake)
	buf.Write(HandshakeFlags)
	buf.Write(HandshakeSenderPID)
	buf.Write(HandshakePeerPID)

	return buf.Bytes()
}

func NewHandshakeRequest() *HandshakeRequest {
	return &HandshakeRequest{}
}
