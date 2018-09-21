package wrappers

import (
	"context"
	"net"
	"time"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/stats"
	"github.com/juju/errors"
)

// ConnPurpose is intended to be identifier of connection purpose. We
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
	connTimeoutRead  = 2 * time.Minute
	connTimeoutWrite = 2 * time.Minute
)

type ioResult struct {
	n   int
	err error
}

type ioFunc func([]byte) (int, error)

// Conn is a basic wrapper for net.Conn providing the most low-level
// logic and management as possible.
type Conn struct {
	conn   net.Conn
	ctx    context.Context
	cancel context.CancelFunc
	connID string
	logger *zap.SugaredLogger

	publicIPv4 net.IP
	publicIPv6 net.IP
}

func (c *Conn) Write(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, errors.Annotate(c.ctx.Err(), "Cannot write because context was closed")
	default:
		n, err := c.doIO(c.conn.Write, p, connTimeoutWrite)

		c.logger.Debugw("Write to stream", "bytes", n, "error", err)
		stats.EgressTraffic(n)

		return n, err
	}
}

func (c *Conn) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, errors.Annotate(c.ctx.Err(), "Cannot read because context was closed")
	default:
		n, err := c.doIO(c.conn.Read, p, connTimeoutRead)

		c.logger.Debugw("Read from stream", "bytes", n, "error", err)
		stats.IngressTraffic(n)

		return n, err
	}
}

func (c *Conn) doIO(callback ioFunc, p []byte, timeout time.Duration) (int, error) {
	resChan := make(chan ioResult, 1)
	timer := time.NewTimer(timeout)

	go func() {
		n, err := callback(p)
		resChan <- ioResult{n: n, err: err}
	}()

	select {
	case res := <-resChan:
		timer.Stop()
		if res.err != nil {
			c.Close() // nolint: gosec
		}
		return res.n, res.err
	case <-c.ctx.Done():
		timer.Stop()
		c.Close() // nolint: gosec
		return 0, errors.Annotate(c.ctx.Err(), "Cannot do IO because context is closed")
	case <-timer.C:
		c.Close() // nolint: gosec
		return 0, errors.Annotate(c.ctx.Err(), "Timeout on IO operation")
	}
}

// Close closes underlying net.Conn instance.
func (c *Conn) Close() error {
	defer c.logger.Debugw("Close connection")

	c.cancel()
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
func NewConn(ctx context.Context, cancel context.CancelFunc, conn net.Conn,
	connID string, purpose ConnPurpose, publicIPv4, publicIPv6 net.IP) StreamReadWriteCloser {
	logger := zap.S().With(
		"connection_id", connID,
		"local_address", conn.LocalAddr(),
		"remote_address", conn.RemoteAddr(),
		"purpose", purpose,
	).Named("conn")

	wrapper := Conn{
		conn:       conn,
		ctx:        ctx,
		cancel:     cancel,
		connID:     connID,
		logger:     logger,
		publicIPv4: publicIPv4,
		publicIPv6: publicIPv6,
	}
	wrapper.logger = logger.With("faked_local_addr", wrapper.LocalAddr())

	return &wrapper
}
