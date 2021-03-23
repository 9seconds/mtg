package mtglib

import (
	"context"
	"net"
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

	return n, err
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

	return n, err
}
