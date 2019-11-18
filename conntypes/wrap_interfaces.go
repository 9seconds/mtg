package conntypes

import (
	"net"

	"go.uber.org/zap"
)

type Wrap interface {
	Conn() net.Conn
	Logger() *zap.SugaredLogger
	LocalAddr() *net.TCPAddr
	RemoteAddr() *net.TCPAddr
}
