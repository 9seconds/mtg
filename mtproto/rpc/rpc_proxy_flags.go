package rpc

import "encoding/binary"

type RPCProxyRequestFlags uint32

const (
	RPCProxyRequestFlagsHasAdTag     RPCProxyRequestFlags = 0x8
	RPCProxyRequestFlagsEncrypted                         = 0x2
	RPCProxyRequestFlagsMagic                             = 0x1000
	RPCProxyRequestFlagsExtMode2                          = 0x20000
	RPCProxyRequestFlagsIntermediate                      = 0x20000000
	RPCProxyRequestFlagsAbdridged                         = 0x40000000
	RPCProxyRequestFlagsQuickAck                          = 0x80000000
)

var rpcProxyRequestFlagsEncryptedPrefix [8]byte

func (r RPCProxyRequestFlags) Bytes() []byte {
	converted := make([]byte, 4)
	binary.LittleEndian.PutUint32(converted, uint32(r))

	return converted
}
