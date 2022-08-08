package testlib

import (
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

type EssentialsConnMock struct {
	mock.Mock
}

func (n *EssentialsConnMock) Read(b []byte) (int, error) {
	args := n.Called(b)

	return args.Int(0), args.Error(1)
}

func (n *EssentialsConnMock) Write(b []byte) (int, error) {
	args := n.Called(b)

	return args.Int(0), args.Error(1)
}

func (n *EssentialsConnMock) Close() error {
	return n.Called().Error(0) //nolint: wrapcheck
}

func (n *EssentialsConnMock) CloseRead() error {
	return n.Called().Error(0) //nolint: wrapcheck
}

func (n *EssentialsConnMock) CloseWrite() error {
	return n.Called().Error(0) //nolint: wrapcheck
}

func (n *EssentialsConnMock) LocalAddr() net.Addr {
	return n.Called().Get(0).(net.Addr) //nolint: forcetypeassert
}

func (n *EssentialsConnMock) RemoteAddr() net.Addr {
	return n.Called().Get(0).(net.Addr) //nolint: forcetypeassert
}

func (n *EssentialsConnMock) SetDeadline(t time.Time) error {
	return n.Called(t).Error(0) //nolint: wrapcheck
}

func (n *EssentialsConnMock) SetReadDeadline(t time.Time) error {
	return n.Called(t).Error(0) //nolint: wrapcheck
}

func (n *EssentialsConnMock) SetWriteDeadline(t time.Time) error {
	return n.Called(t).Error(0) //nolint: wrapcheck
}
