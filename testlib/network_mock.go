package testlib

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/stretchr/testify/mock"
)

type NetworkMock struct {
	mock.Mock
}

func (n *NetworkMock) Dial(network, address string) (net.Conn, error) {
	args := n.Called(network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (n *NetworkMock) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	args := n.Called(ctx, network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (n *NetworkMock) MakeHTTPClient(dialFunc func(ctx context.Context,
	network, address string) (net.Conn, error)) *http.Client {
	return n.Called(dialFunc).Get(0).(*http.Client)
}

func (n *NetworkMock) IdleTimeout() time.Duration {
	return n.Called().Get(0).(time.Duration)
}
