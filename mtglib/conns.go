package mtglib

import (
	"context"
	"fmt"
	"net"
	"time"
)

type connStandard struct {
	conn        net.Conn
	idleTimeout time.Duration
}

func (c connStandard) Read(b []byte) (int, error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, fmt.Errorf("cannot set read deadline: %w", err)
	}

	return c.conn.Read(b)
}

func (c connStandard) Write(b []byte) (int, error) {
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.idleTimeout)); err != nil {
		return 0, fmt.Errorf("cannot set write deadline: %w", err)
	}

	return c.conn.Write(b)
}

func (c connStandard) Close() error {
	return c.conn.Close()
}

func (c connStandard) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c connStandard) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c connStandard) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c connStandard) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c connStandard) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

type connEventTraffic struct {
	connStandard

	connID string
	stream EventStream
	ctx    context.Context
}

func (c connEventTraffic) Read(b []byte) (int, error) {
	n, err := c.connStandard.Read(b)

	if n > 0 {
		c.stream.Send(c.ctx, EventTraffic{
			CreatedAt: time.Now(),
			ConnID:    c.connID,
			Traffic:   uint(n),
			IsRead:    true,
		})
	}

	return n, err
}

func (c connEventTraffic) Write(b []byte) (int, error) {
	n, err := c.connStandard.Write(b)

	if n > 0 {
		c.stream.Send(c.ctx, EventTraffic{
			CreatedAt: time.Now(),
			ConnID:    c.connID,
			Traffic:   uint(n),
			IsRead:    false,
		})
	}

	return n, err
}
