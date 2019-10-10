package rpc

import (
	"encoding/binary"
	"strings"
)

type ProxyRequestFlags uint32

const (
	ProxyRequestFlagsHasAdTag     ProxyRequestFlags = 0x8
	ProxyRequestFlagsEncrypted    ProxyRequestFlags = 0x2
	ProxyRequestFlagsMagic        ProxyRequestFlags = 0x1000
	ProxyRequestFlagsExtMode2     ProxyRequestFlags = 0x20000
	ProxyRequestFlagsIntermediate ProxyRequestFlags = 0x20000000
	ProxyRequestFlagsAbdridged    ProxyRequestFlags = 0x40000000
	ProxyRequestFlagsQuickAck     ProxyRequestFlags = 0x80000000
	ProxyRequestFlagsPad          ProxyRequestFlags = 0x8000000
)

var ProxyRequestFlagsEncryptedPrefix [8]byte

func (r ProxyRequestFlags) Bytes() []byte {
	converted := make([]byte, 4)
	binary.LittleEndian.PutUint32(converted, uint32(r))

	return converted
}

func (r ProxyRequestFlags) String() string {
	flags := make([]string, 0, 7)

	if r&ProxyRequestFlagsHasAdTag != 0 {
		flags = append(flags, "HAS_AD_TAG")
	}
	if r&ProxyRequestFlagsEncrypted != 0 {
		flags = append(flags, "ENCRYPTED")
	}
	if r&ProxyRequestFlagsMagic != 0 {
		flags = append(flags, "MAGIC")
	}
	if r&ProxyRequestFlagsExtMode2 != 0 {
		flags = append(flags, "EXT_MODE_2")
	}
	if r&ProxyRequestFlagsIntermediate != 0 {
		flags = append(flags, "INTERMEDIATE")
	}
	if r&ProxyRequestFlagsAbdridged != 0 {
		flags = append(flags, "ABRIDGED")
	}
	if r&ProxyRequestFlagsQuickAck != 0 {
		flags = append(flags, "QUICK_ACK")
	}
	if r&ProxyRequestFlagsPad != 0 {
		flags = append(flags, "PAD")
	}

	return strings.Join(flags, " | ")
}
