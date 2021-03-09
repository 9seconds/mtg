package network

import (
	"context"
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

type ConnMock struct {
	mock.Mock
}

func (c *ConnMock) Read(b []byte) (int, error) {
	args := c.Called(b)

	return args.Int(0), args.Error(1)
}

func (c *ConnMock) Write(b []byte) (int, error) {
	args := c.Called(b)

	return args.Int(0), args.Error(1)
}

func (c *ConnMock) Close() error {
	return c.Called().Error(0)
}

func (c *ConnMock) LocalAddr() net.Addr {
	return c.Called().Get(0).(net.Addr)
}

func (c *ConnMock) RemoteAddr() net.Addr {
	return c.Called().Get(0).(net.Addr)
}

func (c *ConnMock) SetDeadline(t time.Time) error {
	return c.Called(t).Error(0)
}

func (c *ConnMock) SetReadDeadline(t time.Time) error {
	return c.Called(t).Error(0)
}

func (c *ConnMock) SetWriteDeadline(t time.Time) error {
	return c.Called(t).Error(0)
}

type DialerMock struct {
	mock.Mock
}

func (d *DialerMock) Dial(network, address string) (net.Conn, error) {
	args := d.Called(network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (d *DialerMock) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	args := d.Called(ctx, network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}
