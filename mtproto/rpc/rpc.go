package rpc

const (
	RPCNonceSeqNo     = -2
	RPCHandshakeSeqNo = -1
)

var (
	RPCTagCloseExt     = []byte{0xa2, 0x34, 0xb6, 0x5e}
	RPCTagProxyAns     = []byte{0x0d, 0xda, 0x03, 0x44}
	RPCTagSimpleAck    = []byte{0x9b, 0x40, 0xac, 0x3b}
	RPCTagHandshake    = []byte{0xf5, 0xee, 0x82, 0x76}
	RPCTagNonce        = []byte{0xaa, 0x87, 0xcb, 0x7a}
	RPCTagProxyRequest = []byte{0xee, 0xf1, 0xce, 0x36}

	RPCNonceCryptoAES = []byte{0x01, 0x00, 0x00, 0x00}

	RPCHandshakeFlags = []byte{0x00, 0x00, 0x00, 0x00}

	RPCProxyRequestExtraSize = []byte{0x18, 0x00, 0x00, 0x00}
	RPCProxyRequestProxyTag  = []byte{0xae, 0x26, 0x1e, 0xdb}

	RPCHandshakeSenderPID = []byte{}
	RPCHandshakePeerPID   = []byte{}
)

func init() {
	RPCHandshakeSenderPID = []byte("IPIPPRPDTIME")
	RPCHandshakePeerPID = []byte("IPIPPRPDTIME")
}
