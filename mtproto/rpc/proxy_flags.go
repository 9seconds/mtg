package rpc

import "encoding/binary"

type proxyRequestFlags uint32

const (
	proxyRequestFlagsHasAdTag     proxyRequestFlags = 0x8
	proxyRequestFlagsEncrypted                      = 0x2
	proxyRequestFlagsMagic                          = 0x1000
	proxyRequestFlagsExtMode2                       = 0x20000
	proxyRequestFlagsIntermediate                   = 0x20000000
	proxyRequestFlagsAbdridged                      = 0x40000000
	proxyRequestFlagsQuickAck                       = 0x80000000
)

var proxyRequestFlagsEncryptedPrefix [8]byte

func (r proxyRequestFlags) Bytes() []byte {
	converted := make([]byte, 4)
	binary.LittleEndian.PutUint32(converted, uint32(r))

	return converted
}
