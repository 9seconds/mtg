package testlib

import (
	"context"
	"net/http"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/stretchr/testify/mock"
)

type MtglibNetworkMock struct {
	mock.Mock
}

func (m *MtglibNetworkMock) Dial(network, address string) (essentials.Conn, error) {
	args := m.Called(network, address)

	return args.Get(0).(essentials.Conn), args.Error(1) //nolint: wrapcheck, forcetypeassert
}

func (m *MtglibNetworkMock) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	args := m.Called(ctx, network, address)

	return args.Get(0).(essentials.Conn), args.Error(1) //nolint: wrapcheck, forcetypeassert
}

func (m *MtglibNetworkMock) MakeHTTPClient(dialFunc func(ctx context.Context,
	network, address string) (essentials.Conn, error),
) *http.Client {
	return m.Called(dialFunc).Get(0).(*http.Client) //nolint: forcetypeassert
}
