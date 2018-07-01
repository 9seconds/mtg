package rpc

import "bytes"

const (
	rpcHandshakeTagLength       = 4
	rpcHandshakeFlagsLength     = 4
	rpcHandshakeSenderPIDLength = 12
	rpcHandshakePeerPIDLength   = rpcHandshakeSenderPIDLength

	rpcHandshakeRequestLength = rpcHandshakeTagLength + rpcHandshakeFlagsLength +
		rpcHandshakeSenderPIDLength + rpcHandshakePeerPIDLength
)

var (
	rpcHandshakeSenderPID [rpcHandshakeSenderPIDLength]byte
	rpcHandshakePeerPID   [rpcHandshakePeerPIDLength]byte

	rpcHandshakeTag   = [rpcHandshakeTagLength]byte{0xf5, 0xee, 0x82, 0x76}
	rpcHandshakeFlags = [rpcHandshakeFlagsLength]byte{0x00, 0x00, 0x00, 0x00}

	rpcHandshakeBuffer *bytes.Buffer
)

type RPCHandshakeRequest struct {
}

func (r *RPCHandshakeRequest) Bytes() *bytes.Buffer {
	buf := &bytes.Buffer{}
	buf.Grow(rpcHandshakeRequestLength)

	buf.Write(rpcHandshakeTag[:])
	buf.Write(rpcHandshakeFlags[:])
	buf.Write(rpcHandshakeSenderPID[:])
	buf.Write(rpcHandshakePeerPID[:])

	return buf
}

func init() {
	copy(rpcHandshakeSenderPID[:], "IPIPPRPDTIME")
	copy(rpcHandshakePeerPID[:], "IPIPPRPDTIME")
}

func NewRPCHandshakeRequest() *RPCHandshakeRequest {
	return &RPCHandshakeRequest{}
}
