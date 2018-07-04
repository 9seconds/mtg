package wrappers

import (
	"net"

	"go.uber.org/zap"
)

// LogReadWriteCloser adds additional logging for reading/writing. All
// logging is performed for debug mode only.
type LogReadWriteCloserWithAddr struct {
	conn   ReadWriteCloserWithAddr
	logger *zap.SugaredLogger
	sockid string
	name   string
}

// Read reads from connection
func (l *LogReadWriteCloserWithAddr) Read(p []byte) (n int, err error) {
	n, err = l.conn.Read(p)
	l.logger.Debugw("Finish reading", "name", l.name, "socketid", l.sockid, "nbytes", n, "error", err)
	return
}

// Write writes into connection.
func (l *LogReadWriteCloserWithAddr) Write(p []byte) (n int, err error) {
	n, err = l.conn.Write(p)
	l.logger.Debugw("Finish writing", "name", l.name, "socketid", l.sockid, "nbytes", n, "error", err)
	return
}

// Close closes underlying connection.
func (l *LogReadWriteCloserWithAddr) Close() error {
	err := l.conn.Close()
	l.logger.Debugw("Finish closing socket", "name", l.name, "socketid", l.sockid, "error", err)
	return err
}

func (l *LogReadWriteCloserWithAddr) LocalAddr() *net.TCPAddr {
	return l.conn.LocalAddr()
}

func (l *LogReadWriteCloserWithAddr) RemoteAddr() *net.TCPAddr {
	return l.conn.RemoteAddr()
}

// NewLogRWC wraps ReadWriteCloser with logger calls.
func NewLogRWC(conn ReadWriteCloserWithAddr, logger *zap.SugaredLogger, sockid string, name string) ReadWriteCloserWithAddr {
	return &LogReadWriteCloserWithAddr{
		conn:   conn,
		logger: logger,
		sockid: sockid,
		name:   name,
	}
}
