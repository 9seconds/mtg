package network

import (
	"context"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/stretchr/testify/mock"
)

type DialerMock struct {
	mock.Mock
}

func (d *DialerMock) Dial(network, address string) (essentials.Conn, error) {
	args := d.Called(network, address)

	return args.Get(0).(essentials.Conn), args.Error(1) //nolint: wrapcheck, forcetypeassert
}

func (d *DialerMock) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	args := d.Called(ctx, network, address)

	return args.Get(0).(essentials.Conn), args.Error(1) //nolint: wrapcheck, forcetypeassert
}
