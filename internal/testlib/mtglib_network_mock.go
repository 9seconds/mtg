package testlib

import (
	"context"
	"net"
	"net/http"

	"github.com/stretchr/testify/mock"
)

type MtglibNetworkMock struct {
	mock.Mock
}

func (m *MtglibNetworkMock) Dial(network, address string) (net.Conn, error) {
	args := m.Called(network, address)

	return args.Get(0).(net.Conn), args.Error(1) // nolint: wrapcheck
}

func (m *MtglibNetworkMock) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	args := m.Called(ctx, network, address)

	return args.Get(0).(net.Conn), args.Error(1) // nolint: wrapcheck
}

func (m *MtglibNetworkMock) MakeHTTPClient(dialFunc func(ctx context.Context,
	network, address string) (net.Conn, error)) *http.Client {
	return m.Called(dialFunc).Get(0).(*http.Client)
}
