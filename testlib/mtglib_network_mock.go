package testlib

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/stretchr/testify/mock"
)

type MtglibNetworkMock struct {
	mock.Mock
}

func (n *MtglibNetworkMock) Dial(network, address string) (net.Conn, error) {
	args := n.Called(network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (n *MtglibNetworkMock) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	args := n.Called(ctx, network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (m *MtglibNetworkMock) MakeHTTPClient(dialFunc func(ctx context.Context,
	network, address string) (net.Conn, error)) *http.Client {
	return m.Called(dialFunc).Get(0).(*http.Client)
}

func (m *MtglibNetworkMock) IdleTimeout() time.Duration {
	return m.Called().Get(0).(time.Duration)
}