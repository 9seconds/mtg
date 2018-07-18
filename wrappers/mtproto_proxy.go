package wrappers

import (
	"bytes"
	"fmt"
	"net"

	"github.com/juju/errors"
	"go.uber.org/zap"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
)

// MTProtoProxy is a wrapper which creates/reads RPC responses from Telegram.
type MTProtoProxy struct {
	conn   PacketReadWriteCloser
	req    *rpc.ProxyRequest
	logger *zap.SugaredLogger

	readCounter  uint32
	writeCounter uint32
}

func (m *MTProtoProxy) Read() ([]byte, error) {
	defer func() {
		m.readCounter++
	}()

	m.logger.Debugw("Read packet",
		"counter", m.readCounter,
		"simple_ack", m.req.Options.WriteHacks.SimpleAck,
		"quick_ack", m.req.Options.WriteHacks.QuickAck,
	)

	packet, err := m.conn.Read()
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read packet")
	}

	m.logger.Debugw("Read packet length",
		"counter", m.readCounter,
		"simple_ack", m.req.Options.WriteHacks.SimpleAck,
		"quick_ack", m.req.Options.WriteHacks.QuickAck,
		"length", len(packet),
	)

	if len(packet) < 4 {
		return nil, errors.Annotate(err, "Incorrect packet length")
	}

	tag, packet := packet[:4], packet[4:]
	switch {
	case bytes.Equal(tag, rpc.TagProxyAns):
		return m.readProxyAns(packet)
	case bytes.Equal(tag, rpc.TagSimpleAck):
		return m.readSimpleAck(packet)
	case bytes.Equal(tag, rpc.TagCloseExt):
		return m.readCloseExt()
	}

	return nil, errors.Errorf("Unknown RPC answer %v", tag)
}

func (m *MTProtoProxy) readProxyAns(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.Errorf("Incorrect data of proxy answer: %d", len(data))
	}
	data = data[12:]

	m.logger.Debugw("Read RPC_PROXY_ANS",
		"counter", m.readCounter,
		"length", len(data),
	)

	return data, nil
}

func (m *MTProtoProxy) readSimpleAck(data []byte) ([]byte, error) {
	if len(data) != 12 {
		return nil, errors.Errorf("Incorrect data of simple ack: %d", len(data))
	}
	data = data[8:12]
	m.req.Options.WriteHacks.SimpleAck = true

	m.logger.Debugw("Read RPC_SIMPLE_ACK",
		"counter", m.readCounter,
		"length", len(data),
	)

	return data, nil
}

func (m *MTProtoProxy) readCloseExt() ([]byte, error) {
	m.logger.Debugw("Read RPC_CLOSE_EXT", "counter", m.readCounter)

	return nil, errors.New("Connection has been closed remotely by RPC call")
}

func (m *MTProtoProxy) Write(p []byte) (int, error) {
	defer func() {
		m.writeCounter++
	}()

	m.logger.Debugw("Write packet",
		"length", len(p),
		"counter", m.writeCounter,
		"simple_ack", m.req.Options.ReadHacks.SimpleAck,
		"quick_ack", m.req.Options.ReadHacks.QuickAck,
	)

	header, flags := m.req.MakeHeader(p)
	if ce := m.logger.Desugar().Check(zap.DebugLevel, "RPC_PROXY_REQ header"); ce != nil {
		ce.Write(
			zap.Int("length", len(p)),
			zap.Uint32("counter", m.writeCounter),
			zap.Bool("simple_ack", m.req.Options.ReadHacks.QuickAck),
			zap.Bool("quick_ack", m.req.Options.ReadHacks.SimpleAck),
			zap.String("header", fmt.Sprintf("%v", header.Bytes())),
			zap.Stringer("flags", flags),
		)
	}
	header.Write(p)

	if _, err := m.conn.Write(header.Bytes()); err != nil {
		return 0, err
	}

	return len(p), nil
}

// Logger returns an instance of the logger for this wrapper.
func (m *MTProtoProxy) Logger() *zap.SugaredLogger {
	return m.logger
}

// LocalAddr returns local address of the underlying net.Conn.
func (m *MTProtoProxy) LocalAddr() *net.TCPAddr {
	return m.conn.LocalAddr()
}

// RemoteAddr returns remote address of the underlying net.Conn.
func (m *MTProtoProxy) RemoteAddr() *net.TCPAddr {
	return m.conn.RemoteAddr()
}

// Close closes underlying net.Conn instance.
func (m *MTProtoProxy) Close() error {
	return m.conn.Close()
}

// NewMTProtoProxy creates new RPC wrapper.
func NewMTProtoProxy(conn PacketReadWriteCloser, connOpts *mtproto.ConnectionOpts,
	adTag []byte) (PacketReadWriteCloser, error) {
	req, err := rpc.NewProxyRequest(connOpts.ClientAddr, conn.LocalAddr(), connOpts, adTag)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create new RPC proxy request")
	}

	return &MTProtoProxy{
		conn:   conn,
		logger: conn.Logger().Named("mtproto-proxy"),
		req:    req,
	}, nil
}
