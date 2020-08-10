package rpc

// SeqNo* is the number of the sequence which have special meaning for
// the Telegram.
const (
	SeqNoNonce     = -2
	SeqNoHandshake = -1
)

// Different constants for RPC protocol.
var (
	TagCloseExt     = []byte{0xa2, 0x34, 0xb6, 0x5e}
	TagProxyAns     = []byte{0x0d, 0xda, 0x03, 0x44}
	TagSimpleAck    = []byte{0x9b, 0x40, 0xac, 0x3b}
	TagHandshake    = []byte{0xf5, 0xee, 0x82, 0x76}
	TagNonce        = []byte{0xaa, 0x87, 0xcb, 0x7a}
	TagProxyRequest = []byte{0xee, 0xf1, 0xce, 0x36}

	NonceCryptoAES = []byte{0x01, 0x00, 0x00, 0x00}

	HandshakeFlags = []byte{0x00, 0x00, 0x00, 0x00}

	ProxyRequestExtraSize = []byte{0x18, 0x00, 0x00, 0x00}
	ProxyRequestProxyTag  = []byte{0xae, 0x26, 0x1e, 0xdb}

	HandshakeSenderPID = []byte("IPIPPRPDTIME")
	HandshakePeerPID   = []byte("IPIPPRPDTIME")
)
