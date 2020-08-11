package tlstypes

import (
	"bytes"
	"io"

	"github.com/9seconds/mtg/utils"
)

type Handshake struct {
	Type      HandshakeType
	Version   Version
	Random    [32]byte
	SessionID []byte
	Tail      Byter
}

func (h *Handshake) WriteBytes(writer io.Writer) {
	packetBuf := bytes.Buffer{}

	writer.Write([]byte{byte(h.Type)}) // nolint: errcheck

	packetBuf.Write(h.Version.Bytes())
	packetBuf.Write(h.Random[:])
	packetBuf.WriteByte(byte(len(h.SessionID)))
	packetBuf.Write(h.SessionID)
	h.Tail.WriteBytes(&packetBuf)

	sizeUint24 := utils.ToUint24(uint32(packetBuf.Len()))
	sizeUint24Bytes := sizeUint24[:]
	sizeUint24Bytes[0], sizeUint24Bytes[2] = sizeUint24Bytes[2], sizeUint24Bytes[0]

	writer.Write(sizeUint24Bytes) // nolint: errcheck
	packetBuf.WriteTo(writer)     // nolint: errcheck
}

func (h *Handshake) Len() int {
	buf := bytes.Buffer{}

	h.WriteBytes(&buf)

	return buf.Len()
}
