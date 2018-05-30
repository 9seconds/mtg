package server

import (
	"io"

	"go.uber.org/zap"
)

type LogReadWriteCloser struct {
	conn   io.ReadWriteCloser
	logger *zap.SugaredLogger
	sockid string
	name   string
}

func (l *LogReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = l.conn.Read(p)
	l.logger.Debugw("Finish reading", "name", l.name, "socketid", l.sockid, "nbytes", n, "error", err)
	return
}

func (l *LogReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = l.conn.Write(p)
	l.logger.Debugw("Finish writing", "name", l.name, "socketid", l.sockid, "nbytes", n, "error", err)
	return
}

func (l *LogReadWriteCloser) Close() error {
	err := l.conn.Close()
	l.logger.Debugw("Finish closing socket", "name", l.name, "socketid", l.sockid, "error", err)
	return err
}

func newLogReadWriteCloser(conn io.ReadWriteCloser, logger *zap.SugaredLogger, sockid string, name string) io.ReadWriteCloser {
	return &LogReadWriteCloser{
		conn:   conn,
		logger: logger,
		sockid: sockid,
		name:   name,
	}
}
