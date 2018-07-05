package rpc

import "bytes"

type RPCHandshakeRequest struct {
}

func (r *RPCHandshakeRequest) Bytes() []byte {
	buf := &bytes.Buffer{}

	buf.Write(RPCTagHandshake)
	buf.Write(RPCHandshakeFlags)
	buf.Write(RPCHandshakeSenderPID)
	buf.Write(RPCHandshakePeerPID)

	return buf.Bytes()
}

func NewRPCHandshakeRequest() *RPCHandshakeRequest {
	return &RPCHandshakeRequest{}
}
