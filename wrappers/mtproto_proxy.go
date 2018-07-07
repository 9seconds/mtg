package wrappers

import (
	"bytes"
	"net"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/mtproto/rpc"
)

type MTProtoProxy struct {
	conn WrapPacketReadWriteCloser
	req  *rpc.ProxyRequest

	readCounter  uint32
	writeCounter uint32
}

func (m *MTProtoProxy) Read() ([]byte, error) {
	m.LogDebug("Read packet",
		"counter", m.readCounter,
		"simple_ack", m.req.Options.WriteHacks.SimpleAck,
		"quick_ack", m.req.Options.WriteHacks.QuickAck,
	)

	packet, err := m.conn.Read()
	if err != nil {
		return nil, errors.Annotate(err, "Cannot read packet")
	}
	m.LogDebug("Read packet length",
		"counter", m.readCounter,
		"simple_ack", m.req.Options.WriteHacks.SimpleAck,
		"quick_ack", m.req.Options.WriteHacks.QuickAck,
		"length", len(packet),
	)

	if len(packet) < 4 {
		return nil, errors.Annotate(err, "Incorrect packet length")
	}

	tag, packet := packet[:4], packet[4:]
	m.LogDebug("Read RPC tag",
		"counter", m.readCounter,
		"simple_ack", m.req.Options.WriteHacks.SimpleAck,
		"quick_ack", m.req.Options.WriteHacks.QuickAck,
		"tag", tag,
	)

	m.readCounter++
	switch {
	case bytes.Equal(tag, rpc.TagProxyAns):
		return m.readProxyAns(packet)
	case bytes.Equal(tag, rpc.TagSimpleAck):
		return m.readSimpleAck(packet)
	case bytes.Equal(tag, rpc.TagCloseExt):
		return m.readCloseExt(packet)
	}

	return nil, errors.Errorf("Unknown RPC answer %v", tag)
}

func (m *MTProtoProxy) readProxyAns(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.Errorf("Incorrect data of proxy answer: %d", len(data))
	}

	return data[12:], nil
}

func (m *MTProtoProxy) readSimpleAck(data []byte) ([]byte, error) {
	if len(data) != 12 {
		return nil, errors.Errorf("Incorrect data of simple ack: %d", len(data))
	}

	return data[8:12], nil // 0:8 - connection id
}

func (m *MTProtoProxy) readCloseExt(data []byte) ([]byte, error) {
	return nil, errors.New("Connection has been closed remotely by RPC call")
}

func (m *MTProtoProxy) Write(p []byte) (int, error) {
	m.LogDebug("Write packet",
		"length", len(p),
		"counter", m.writeCounter,
		"simple_ack", m.req.Options.ReadHacks.SimpleAck,
		"quick_ack", m.req.Options.ReadHacks.QuickAck,
	)
	m.writeCounter++

	if _, err := m.conn.Write(p); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (m *MTProtoProxy) LogDebug(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "proxy"})
	m.conn.LogDebug(msg, data...)
}

func (m *MTProtoProxy) LogInfo(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "proxy"})
	m.conn.LogInfo(msg, data...)
}

func (m *MTProtoProxy) LogWarn(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "proxy"})
	m.conn.LogWarn(msg, data...)
}

func (m *MTProtoProxy) LogError(msg string, data ...interface{}) {
	data = append(data, []interface{}{"type", "proxy"})
	m.conn.LogError(msg, data...)
}

func (m *MTProtoProxy) LocalAddr() *net.TCPAddr {
	return m.conn.LocalAddr()
}

func (m *MTProtoProxy) RemoteAddr() *net.TCPAddr {
	return m.conn.RemoteAddr()
}

func (m *MTProtoProxy) Close() error {
	return m.conn.Close()
}

func NewMTProtoProxy(conn WrapPacketReadWriteCloser, connOpts *mtproto.ConnectionOpts, adTag []byte) (WrapPacketReadWriteCloser, error) {
	req, err := rpc.NewProxyRequest(connOpts.ClientAddr, conn.LocalAddr(), connOpts, adTag)
	if err != nil {
		return nil, errors.Annotate(err, "Cannot create new RPC proxy request")
	}

	return &MTProtoProxy{
		conn: conn,
		req:  req,
	}, nil
}
