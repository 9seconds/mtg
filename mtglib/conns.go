package mtglib

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"time"
)

type connTelegramTraffic struct {
	net.Conn

	connID string
	stream EventStream
	ctx    context.Context
}

func (c connTelegramTraffic) Read(b []byte) (int, error) {
	n, err := c.Conn.Read(b)

	if n > 0 {
		c.stream.Send(c.ctx, EventTraffic{
			CreatedAt: time.Now(),
			ConnID:    c.connID,
			Traffic:   uint(n),
			IsRead:    true,
		})
	}

	return n, err // nolint: wrapcheck
}

func (c connTelegramTraffic) Write(b []byte) (int, error) {
	n, err := c.Conn.Write(b)

	if n > 0 {
		c.stream.Send(c.ctx, EventTraffic{
			CreatedAt: time.Now(),
			ConnID:    c.connID,
			Traffic:   uint(n),
			IsRead:    false,
		})
	}

	return n, err // nolint: wrapcheck
}

type connRewind struct {
	net.Conn

	active io.Reader
	buf    bytes.Buffer
	mutex  sync.RWMutex
}

func (c *connRewind) Read(p []byte) (int, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	return c.active.Read(p)
}

func (c *connRewind) Rewind() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.active = io.MultiReader(&c.buf, c.Conn)
}

func newConnRewind(conn net.Conn) *connRewind {
	rv := &connRewind{
		Conn: conn,
	}
	rv.active = io.TeeReader(conn, &rv.buf)

	return rv
}
