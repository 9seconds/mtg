package mtglib

import (
	"bytes"
	"context"
	"io"

	"github.com/9seconds/mtg/v2/essentials"
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
