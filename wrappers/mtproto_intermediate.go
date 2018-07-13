package wrappers

import (
	"bytes"
	"encoding/binary"
	"io"
	"net"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/mtproto"
)

const mtprotoIntermediateQuickAckLength = 0x80000000

// MTProtoIntermediate presents intermediate connection between client
// and Telegram.
type MTProtoIntermediate struct {
	conn   StreamReadWriteCloser
	opts   *mtproto.ConnectionOpts
	logger *zap.SugaredLogger

	readCounter  uint32
	writeCounter uint32
}

func (m *MTProtoIntermediate) Read() ([]byte, error) {
	defer func() {
		m.readCounter++
	}()

	m.logger.Debugw("Read packet",
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

	m.logger.Debugw("Packet message length",
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

	return buf.Bytes()[:length], nil
}

func (m *MTProtoIntermediate) Write(p []byte) (int, error) {
	defer func() {
		m.writeCounter++
	}()

	m.logger.Debugw("Write packet",
		"simple_ack", m.opts.WriteHacks.SimpleAck,
		"quick_ack", m.opts.WriteHacks.QuickAck,
		"counter", m.writeCounter,
	)

	if m.opts.WriteHacks.SimpleAck {
		return m.conn.Write(p)
	}

	var length [4]byte
	binary.LittleEndian.PutUint32(length[:], uint32(len(p)))

	return m.conn.Write(append(length[:], p...))
}

// Logger returns an instance of the logger for this wrapper.
func (m *MTProtoIntermediate) Logger() *zap.SugaredLogger {
	return m.logger
}

// LocalAddr returns local address of the underlying net.Conn.
func (m *MTProtoIntermediate) LocalAddr() *net.TCPAddr {
	return m.conn.LocalAddr()
}

// RemoteAddr returns remote address of the underlying net.Conn.
func (m *MTProtoIntermediate) RemoteAddr() *net.TCPAddr {
	return m.conn.RemoteAddr()
}

// Close closes underlying net.Conn instance.
func (m *MTProtoIntermediate) Close() error {
	return m.conn.Close()
}

// NewMTProtoIntermediate creates new PacketWrapper for intermediate
// client connection.
func NewMTProtoIntermediate(conn StreamReadWriteCloser, opts *mtproto.ConnectionOpts) PacketReadWriteCloser {
	return &MTProtoIntermediate{
		conn:   conn,
		logger: conn.Logger().Named("mtproto-intermediate"),
		opts:   opts,
	}
}
