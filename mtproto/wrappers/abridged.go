package wrappers

import (
	"bytes"
	"io"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

type uint24 [3]byte

const (
	abridgedSmallPacketLength = 0x7f
	abridgedQuickAckLength    = 0x80
	abridgedLargePacketLength = 16777216 // 256 ^ 3
)

type AbridgedReadWriteCloserWithAddr struct {
	wrappers.BufferedReader

	conn wrappers.ReadWriteCloserWithAddr
	opts *mtproto.ConnectionOpts
}

func (a *AbridgedReadWriteCloserWithAddr) Read(p []byte) (int, error) {
	return a.BufferedRead(p, func() error {
		a.opts.QuickAck = false
		a.opts.SimpleAck = false

		buf := &bytes.Buffer{}
		buf.Grow(3)

		if _, err := io.CopyN(buf, a.conn, 1); err != nil {
			return errors.Annotate(err, "Cannot read message length")
		}
		msgLength := uint8(buf.Bytes()[0])
		buf.Reset()

		if msgLength >= abridgedQuickAckLength {
			a.opts.QuickAck = true
			msgLength -= 0x80
		}

		msgLength32 := uint32(msgLength)
		if msgLength == abridgedSmallPacketLength {
			if _, err := io.CopyN(buf, a.conn, 3); err != nil {
				return errors.Annotate(err, "Cannot read the correct message length")
			}
			number := uint24{}
			copy(number[:], buf.Bytes())
			msgLength32 = fromUint24(number)
		}
		msgLength32 *= 4

		if _, err := io.CopyN(a.Buffer, a.conn, int64(msgLength32)); err != nil {
			return errors.Annotate(err, "Cannot read message")
		}

		return nil
	})
}

func (a *AbridgedReadWriteCloserWithAddr) Write(p []byte) (int, error) {
	if len(p)%4 != 0 {
		return 0, errors.Errorf("Incorrect packet length %d", len(p))
	}
	if a.opts.SimpleAck {
		return a.conn.Write(reverseBytes(p))
	}

	packetLength := len(p) / 4
	switch {
	case packetLength < abridgedSmallPacketLength:
		newData := append([]byte{byte(packetLength)}, p...)
		return a.conn.Write(newData)

	case packetLength < abridgedLargePacketLength:
		length24 := toUint24(uint32(packetLength))
		buf := &bytes.Buffer{}
		buf.Grow(1 + 3 + len(p))
		buf.WriteByte(byte(abridgedSmallPacketLength))
		buf.Write(length24[:])
		buf.Write(p)
		return a.conn.Write(buf.Bytes())

	default:
		return 0, errors.Errorf("Packet is too big %d", len(p))
	}
}

func (a *AbridgedReadWriteCloserWithAddr) Close() error {
	return a.conn.Close()
}

func (a *AbridgedReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return a.conn.LocalAddr()
}

func (a *AbridgedReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return a.conn.RemoteAddr()
}

func toUint24(number uint32) uint24 {
	return uint24{byte(number), byte(number >> 8), byte(number >> 16)}
}

func fromUint24(number uint24) uint32 {
	return uint32(number[0]) + (uint32(number[1]) << 8) + (uint32(number[2]) << 16)
}

func NewAbridgedRWC(conn wrappers.ReadWriteCloserWithAddr, connOpts *mtproto.ConnectionOpts) wrappers.ReadWriteCloserWithAddr {
	return &AbridgedReadWriteCloserWithAddr{
		BufferedReader: wrappers.NewBufferedReader(),
		conn:           conn,
		opts:           connOpts,
	}
}
