package tlstypes

import (
	"bytes"

	"mtg/utils"
)

type Handshake struct {
	Type      HandshakeType
	Version   Version
	Random    [32]byte
	SessionID []byte
	Tail      Byter
}

func (h *Handshake) Bytes() []byte {
	buf := bytes.Buffer{}
	packetBuf := bytes.Buffer{}

	buf.WriteByte(byte(h.Type))

	packetBuf.Write(h.Version.Bytes())
	packetBuf.Write(h.Random[:])
	packetBuf.WriteByte(byte(len(h.SessionID)))
	packetBuf.Write(h.SessionID)
	packetBuf.Write(h.Tail.Bytes())

	sizeUint24 := utils.ToUint24(uint32(packetBuf.Len()))
	sizeUint24Bytes := sizeUint24[:]
	sizeUint24Bytes[0], sizeUint24Bytes[2] = sizeUint24Bytes[2], sizeUint24Bytes[0]

	buf.Write(sizeUint24Bytes)
	packetBuf.WriteTo(&buf) // nolint: errcheck

	return buf.Bytes()
}
