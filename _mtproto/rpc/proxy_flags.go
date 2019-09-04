package rpc

import (
	"encoding/binary"
	"strings"
)

type proxyRequestFlags uint32

const (
	proxyRequestFlagsHasAdTag     proxyRequestFlags = 0x8
	proxyRequestFlagsEncrypted    proxyRequestFlags = 0x2
	proxyRequestFlagsMagic        proxyRequestFlags = 0x1000
	proxyRequestFlagsExtMode2     proxyRequestFlags = 0x20000
	proxyRequestFlagsIntermediate proxyRequestFlags = 0x20000000
	proxyRequestFlagsAbdridged    proxyRequestFlags = 0x40000000
	proxyRequestFlagsQuickAck     proxyRequestFlags = 0x80000000
	proxyRequestFlagsPad          proxyRequestFlags = 0x8000000
)

var proxyRequestFlagsEncryptedPrefix [8]byte

func (r proxyRequestFlags) Bytes() []byte {
	converted := make([]byte, 4)
	binary.LittleEndian.PutUint32(converted, uint32(r))

	return converted
}

func (r proxyRequestFlags) String() string {
	flags := make([]string, 0, 7)

	if r&proxyRequestFlagsHasAdTag != 0 {
		flags = append(flags, "HAS_AD_TAG")
	}
	if r&proxyRequestFlagsEncrypted != 0 {
		flags = append(flags, "ENCRYPTED")
	}
	if r&proxyRequestFlagsMagic != 0 {
		flags = append(flags, "MAGIC")
	}
	if r&proxyRequestFlagsExtMode2 != 0 {
		flags = append(flags, "EXT_MODE_2")
	}
	if r&proxyRequestFlagsIntermediate != 0 {
		flags = append(flags, "INTERMEDIATE")
	}
	if r&proxyRequestFlagsAbdridged != 0 {
		flags = append(flags, "ABRIDGED")
	}
	if r&proxyRequestFlagsQuickAck != 0 {
		flags = append(flags, "QUICK_ACK")
	}
	if r&proxyRequestFlagsPad != 0 {
		flags = append(flags, "PAD")
	}

	return strings.Join(flags, " | ")
}
