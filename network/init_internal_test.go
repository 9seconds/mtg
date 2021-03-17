package network

import (
	"context"
	"net"

	"github.com/stretchr/testify/mock"
)

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
