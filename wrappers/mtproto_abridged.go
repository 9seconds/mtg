package wrappers

import (
	"bytes"
	"io"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/utils"
)

const (
	abridgedSmallPacketLength = 0x7f
	abridgedQuickAckLength    = 0x80
	abridgedLargePacketLength = 16777216 // 256 ^ 3
)

type MTProtoAbridged struct {
	conn WrapStreamReadWriteCloser
	opts *mtproto.ConnectionOpts

	readCounter  uint32
	writeCounter uint32
}

func (m *MTProtoAbridged) Read() ([]byte, error) {
	m.LogDebug("Read abridged packet",
		"simple_ack", m.opts.WriteHacks.SimpleAck,
		"quick_ack", m.opts.WriteHacks.QuickAck,
		"counter", m.readCounter,
	)

	buf := &bytes.Buffer{}
	buf.Grow(3)

	if _, err := io.CopyN(buf, m.conn, 1); err != nil {
		return nil, errors.Annotate(err, "Cannot read message length")
	}
	msgLength := uint8(buf.Bytes()[0])
	buf.Reset()

	m.LogDebug("Abridged packet first byte",
		"byte", msgLength,
		"counter", m.readCounter,
	)

	if msgLength >= abridgedQuickAckLength {
		m.opts.ReadHacks.QuickAck = true
		msgLength -= abridgedQuickAckLength
	}

	msgLength32 := uint32(msgLength)
	if msgLength == abridgedSmallPacketLength {
		if _, err := io.CopyN(buf, m.conn, 3); err != nil {
			return nil, errors.Annotate(err, "Cannot read the correct message length")
		}
		number := utils.Uint24{}
		copy(number[:], buf.Bytes())
		msgLength32 = utils.FromUint24(number)
	}
	msgLength32 *= 4

	m.LogDebug("Abridged packet length",
		"length", msgLength32,
		"counter", m.readCounter,
	)

	buf.Reset()
	buf.Grow(int(msgLength32))

	if _, err := io.CopyN(buf, m.conn, int64(msgLength32)); err != nil {
		return nil, errors.Annotate(err, "Cannot read message")
	}
	m.readCounter++

	return buf.Bytes(), nil
}

func (m *MTProtoAbridged) Write(p []byte) (int, error) {
	m.LogDebug("Write abridged packet",
		"length", len(p),
		"simple_ack", m.opts.WriteHacks.SimpleAck,
		"quick_ack", m.opts.WriteHacks.QuickAck,
		"counter", m.writeCounter,
	)

	if len(p)%4 == 0 {
		return 0, errors.Errorf("Incorrect packet length %d", len(p))
	}

	if m.opts.WriteHacks.SimpleAck {
		return m.conn.Write(utils.ReverseBytes(p))
	}

	packetLength := len(p) / 4
	switch {
	case packetLength < abridgedSmallPacketLength:
		newData := append([]byte{byte(packetLength)}, p...)

		m.writeCounter++
		return m.conn.Write(newData)

	case packetLength < abridgedLargePacketLength:
		length24 := utils.ToUint24(uint32(packetLength))

		buf := &bytes.Buffer{}
		buf.Grow(1 + 3 + len(p))

		buf.WriteByte(byte(abridgedSmallPacketLength))
		buf.Write(length24[:])
		buf.Write(p)

		m.writeCounter++
		return m.conn.Write(buf.Bytes())
	}

	return 0, errors.Errorf("Packet is too big %d", len(p))
}

func (m *MTProtoAbridged) LogDebug(msg string, data ...interface{}) {
	m.conn.LogDebug(msg, data...)
}

func (m *MTProtoAbridged) LogInfo(msg string, data ...interface{}) {
	m.conn.LogInfo(msg, data...)
}

func (m *MTProtoAbridged) LogWarn(msg string, data ...interface{}) {
	m.conn.LogWarn(msg, data...)
}

func (m *MTProtoAbridged) LogError(msg string, data ...interface{}) {
	m.conn.LogError(msg, data...)
}

func (m *MTProtoAbridged) LocalAddr() *net.TCPAddr {
	return m.conn.LocalAddr()
}

func (m *MTProtoAbridged) RemoteAddr() *net.TCPAddr {
	return m.conn.RemoteAddr()
}

func (m *MTProtoAbridged) Close() error {
	return m.conn.Close()
}

func NewMTProtoAbridged(conn WrapStreamReadWriteCloser, opts *mtproto.ConnectionOpts) WrapPacketReadWriteCloser {
	return &MTProtoAbridged{
		conn: conn,
		opts: opts,
	}
}
