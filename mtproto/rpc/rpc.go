package rpc

import "bytes"

const (
	RPCNonceSeqNo     = -2
	RPCHandshakeSeqNo = -1
)

type RPC interface {
	Bytes() *bytes.Buffer
}
