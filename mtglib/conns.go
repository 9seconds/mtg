package mtglib

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/dolonet/mtg-multi/essentials"
	"github.com/pires/go-proxyproto"
)

type connTraffic struct {
	essentials.Conn

	streamID string
	stream   EventStream
	ctx      context.Context
}

func (c connTraffic) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)

	if n > 0 {
		c.stream.Send(c.ctx, NewEventTraffic(c.streamID, uint(n), true))
	}

	return n, err //nolint: wrapcheck
}

func (c connTraffic) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)

	if n > 0 {
		c.stream.Send(c.ctx, NewEventTraffic(c.streamID, uint(n), false))
	}

	return n, err //nolint: wrapcheck
}

type connRewind struct {
	essentials.Conn

	buf    bytes.Buffer
	active io.Reader
}

func (c *connRewind) Read(p []byte) (int, error) {
	return c.active.Read(p)
}

func (c *connRewind) Rewind() {
	c.active = io.MultiReader(&c.buf, c.Conn)
}

func newConnRewind(conn essentials.Conn) *connRewind {
	rv := &connRewind{
		Conn: conn,
	}
	rv.active = io.TeeReader(conn, &rv.buf)

	return rv
}

type connProxyProtocol struct {
	essentials.Conn

	sourceAddr     net.Addr
	headersWritten bool
}

func (c *connProxyProtocol) Write(p []byte) (int, error) {
	if !c.headersWritten {
		headers := proxyproto.HeaderProxyFromAddrs(2, c.sourceAddr, c.RemoteAddr())

		toSend, err := headers.Format()
		if err != nil {
			panic(err)
		}

		if _, err := c.Conn.Write(toSend); err != nil {
			return 0, fmt.Errorf("cannot send proxy protocol header: %w", err)
		}

		c.headersWritten = true
	}

	return c.Conn.Write(p)
}

func newConnProxyProtocol(source, target essentials.Conn) *connProxyProtocol {
	return &connProxyProtocol{
		Conn:       target,
		sourceAddr: source.RemoteAddr(),
	}
}

// idleTracker is a shared idle tracker for a pair of relay connections.
// Both directions update the same timestamp so that activity in one direction
// prevents the other (idle) direction from timing out.
type idleTracker struct {
	lastActive atomic.Pointer[time.Time]
	timeout    time.Duration
}

func newIdleTracker(timeout time.Duration) *idleTracker {
	t := &idleTracker{timeout: timeout}
	t.touch()

	return t
}

func (t *idleTracker) touch() {
	stamp := time.Now()
	t.lastActive.Store(&stamp)
}

func (t *idleTracker) isIdle() bool {
	return time.Since(*t.lastActive.Load()) >= t.timeout
}

type connIdleTimeout struct {
	essentials.Conn

	tracker *idleTracker
}

func (c connIdleTimeout) Read(b []byte) (int, error) {
	var netErr net.Error

	for {
		c.SetReadDeadline(time.Now().Add(c.tracker.timeout)) //nolint: errcheck

		n, err := c.Conn.Read(b)

		switch {
		case err == nil:
			c.tracker.touch()
			return n, nil
		case errors.As(err, &netErr) && netErr.Timeout() && !c.tracker.isIdle():
			continue
		}

		return n, err
	}
}

func (c connIdleTimeout) Write(b []byte) (int, error) {
	c.SetWriteDeadline(time.Now().Add(c.tracker.timeout)) //nolint: errcheck

	n, err := c.Conn.Write(b)
	if n > 0 {
		c.tracker.touch()
	}

	return n, err //nolint: wrapcheck
}
