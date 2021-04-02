package testlib

import (
	"net"
	"time"

	"github.com/stretchr/testify/mock"
)

type NetConnMock struct {
	mock.Mock
}

func (n *NetConnMock) Read(b []byte) (int, error) {
	args := n.Called(b)

	return args.Int(0), args.Error(1)
}

func (n *NetConnMock) Write(b []byte) (int, error) {
	args := n.Called(b)

	return args.Int(0), args.Error(1)
}

func (n *NetConnMock) Close() error {
	return n.Called().Error(0)
}

func (n *NetConnMock) LocalAddr() net.Addr {
	return n.Called().Get(0).(net.Addr)
}

func (n *NetConnMock) RemoteAddr() net.Addr {
	return n.Called().Get(0).(net.Addr)
}

func (n *NetConnMock) SetDeadline(t time.Time) error {
	return n.Called(t).Error(0)
}

func (n *NetConnMock) SetReadDeadline(t time.Time) error {
	return n.Called(t).Error(0)
}

func (n *NetConnMock) SetWriteDeadline(t time.Time) error {
	return n.Called(t).Error(0)
}
