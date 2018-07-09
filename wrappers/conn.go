package wrappers

import (
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/stats"
)

// ConnPurpose is intented to be identifier of connection purpose. We
// sometimes want to treat client/telegram connection differently (for
// logging for example).
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

// ConnPurpose* define different connection types.
const (
	ConnPurposeClient = iota
	ConnPurposeTelegram
)

const (
	connTimeoutRead  = 5 * time.Minute
	connTimeoutWrite = 5 * time.Minute
)

// Conn is a basic wrapper for net.Conn providing the most low-level
// logic and management as possible.
type Conn struct {
	connID string
	conn   net.Conn
	logger *zap.SugaredLogger

	publicIPv4 net.IP
	publicIPv6 net.IP
}

func (c *Conn) Write(p []byte) (int, error) {
	c.conn.SetWriteDeadline(time.Now().Add(connTimeoutWrite)) // nolint: errcheck
	n, err := c.conn.Write(p)

	c.logger.Debugw("Write to stream", "bytes", n, "error", err)
	stats.EgressTraffic(n)

	return n, err
}

func (c *Conn) Read(p []byte) (int, error) {
	c.conn.SetReadDeadline(time.Now().Add(connTimeoutRead)) // nolint: errcheck
	n, err := c.conn.Read(p)

	c.logger.Debugw("Read from stream", "bytes", n, "error", err)
	stats.IngressTraffic(n)

	return n, err
}

// Close closes underlying net.Conn instance.
func (c *Conn) Close() error {
	defer c.logger.Debugw("Close connection")
	return c.conn.Close()
}

// Logger returns an instance of the logger for this wrapper.
func (c *Conn) Logger() *zap.SugaredLogger {
	return c.logger
}

// LocalAddr returns local address of the underlying net.Conn.
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

// RemoteAddr returns remote address of the underlying net.Conn.
func (c *Conn) RemoteAddr() *net.TCPAddr {
	return c.conn.RemoteAddr().(*net.TCPAddr)
}

// NewConn initializes Conn wrapper for net.Conn.
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
