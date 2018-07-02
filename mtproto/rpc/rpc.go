package rpc

const (
	RPCNonceSeqNo     = -2
	RPCHandshakeSeqNo = -1
)

type Extras struct {
	QuickAck  bool
	SimpleAck bool
}
