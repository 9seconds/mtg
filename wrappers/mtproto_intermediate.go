package wrappers

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"

	"github.com/9seconds/mtg/mtproto"
	"github.com/juju/errors"
)

const mtprotoIntermediateQuickAckLength = 0x80000000

type MTProtoIntermediate struct {
	conn StreamReadWriteCloser
	opts *mtproto.ConnectionOpts

	readCounter  uint32
	writeCounter uint32
}

func (m *MTProtoIntermediate) Read() ([]byte, error) {
	m.LogDebug("Read packet",
		"simple_ack", m.opts.ReadHacks.SimpleAck,
		"quick_ack", m.opts.ReadHacks.QuickAck,
		"counter", m.readCounter,
	)

	buf := &bytes.Buffer{}
	buf.Grow(4)

	if _, err := io.CopyN(buf, m.conn, 4); err != nil {
		return nil, errors.Annotate(err, "Cannot read message length")
	}
	length := binary.LittleEndian.Uint32(buf.Bytes())

	m.LogDebug("Packet message length",
		"simple_ack", m.opts.ReadHacks.SimpleAck,
		"quick_ack", m.opts.ReadHacks.QuickAck,
		"counter", m.readCounter,
		"length", length,
	)

	if length > mtprotoIntermediateQuickAckLength {
		m.opts.ReadHacks.QuickAck = true
		length -= mtprotoIntermediateQuickAckLength
	}

	buf.Reset()
	buf.Grow(int(length))
	if _, err := io.CopyN(buf, m.conn, int64(length)); err != nil {
		return nil, errors.Annotate(err, "Cannot read the message")
	}

	if length%4 != 0 {
		length -= length % 4
	}
	m.readCounter++

	return buf.Bytes()[:length], nil
}

func (m *MTProtoIntermediate) Write(p []byte) (int, error) {
	m.LogDebug("Write packet",
		"simple_ack", m.opts.WriteHacks.SimpleAck,
		"quick_ack", m.opts.WriteHacks.QuickAck,
		"counter", m.writeCounter,
	)
	m.writeCounter++

	if m.opts.ReadHacks.SimpleAck {
		return m.conn.Write(p)
	}

	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(len(p)))

	return m.conn.Write(append(length[:], p...))
}

func (m *MTProtoIntermediate) LogDebug(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "intermediate"}...)
	m.conn.LogDebug(msg, data...)
}

func (m *MTProtoIntermediate) LogInfo(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "intermediate"}...)
	m.conn.LogInfo(msg, data...)
}

func (m *MTProtoIntermediate) LogWarn(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "intermediate"}...)
	m.conn.LogWarn(msg, data...)
}

func (m *MTProtoIntermediate) LogError(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "intermediate"}...)
	m.conn.LogError(msg, data...)
}

func (m *MTProtoIntermediate) LocalAddr() *net.TCPAddr {
	return m.conn.LocalAddr()
}

func (m *MTProtoIntermediate) RemoteAddr() *net.TCPAddr {
	return m.conn.RemoteAddr()
}

func (m *MTProtoIntermediate) Close() error {
	return m.conn.Close()
}

func NewMTProtoIntermediate(conn StreamReadWriteCloser, opts *mtproto.ConnectionOpts) PacketReadWriteCloser {
	return &MTProtoIntermediate{
		conn: conn,
		opts: opts,
	}
}
