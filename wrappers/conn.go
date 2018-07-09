package wrappers

import (
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/stats"
)

type ConnPurpose uint8

func (c ConnPurpose) String() string {
	switch c {
	case ConnPurposeClient:
		return "client"
	case ConnPurposeTelegram:
		return "telegram"
	}

	return ""
}

const (
	ConnPurposeClient = iota
	ConnPurposeTelegram
)

const (
	connTimeoutRead  = 5 * time.Minute
	connTimeoutWrite = 5 * time.Minute
)

type Conn struct {
	connID string
	conn   net.Conn
	logger *zap.SugaredLogger

	publicIPv4 net.IP
	publicIPv6 net.IP
}

func (c *Conn) Write(p []byte) (int, error) {
	c.conn.SetWriteDeadline(time.Now().Add(connTimeoutWrite))
	n, err := c.conn.Write(p)

	c.logger.Debugw("Write to stream", "bytes", n, "error", err)
	stats.EgressTraffic(n)

	return n, err
}

func (c *Conn) Read(p []byte) (int, error) {
	c.conn.SetReadDeadline(time.Now().Add(connTimeoutRead))
	n, err := c.conn.Read(p)

	c.logger.Debugw("Read from stream", "bytes", n, "error", err)
	stats.IngressTraffic(n)

	return n, err
}

func (c *Conn) Close() error {
	defer c.logger.Debugw("Closed connection")
	return c.conn.Close()
}

func (c *Conn) LocalAddr() *net.TCPAddr {
	addr := c.conn.LocalAddr().(*net.TCPAddr)
	newAddr := *addr

	if c.RemoteAddr().IP.To4() != nil {
		if c.publicIPv4 != nil {
			newAddr.IP = c.publicIPv4
		}
	} else if c.publicIPv6 != nil {
		newAddr.IP = c.publicIPv6
	}

	return &newAddr
}

func (c *Conn) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr().(*net.TCPAddr)
}

func (c *Conn) Logger() *zap.SugaredLogger {
	return c.logger
}

func NewConn(conn net.Conn, connID string, purpose ConnPurpose, publicIPv4, publicIPv6 net.IP) StreamReadWriteCloser {
	logger := zap.S().With(
		"connection_id", connID,
		"local_address", conn.LocalAddr(),
		"remote_address", conn.RemoteAddr(),
		"purpose", purpose,
	).Named("conn")

	wrapper := Conn{
		logger:     logger,
		connID:     connID,
		conn:       conn,
		publicIPv4: publicIPv4,
		publicIPv6: publicIPv6,
	}
	wrapper.logger = logger.With("faked_local_addr", wrapper.LocalAddr())

	return &wrapper
}
