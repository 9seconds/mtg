package proxy

import "io"

type TrafficReadWriteCloser struct {
	conn          io.ReadWriteCloser
	readCallback  func(int)
	writeCallback func(int)
}

func (t *TrafficReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = t.conn.Read(p)
	t.readCallback(n)
	return
}

func (t *TrafficReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = t.conn.Write(p)
	t.writeCallback(n)
	return
}

func (t *TrafficReadWriteCloser) Close() error {
	return t.conn.Close()
}

func newTrafficReadWriteCloser(conn io.ReadWriteCloser, readCallback, writeCallback func(int)) io.ReadWriteCloser {
	return &TrafficReadWriteCloser{
		conn:          conn,
		readCallback:  readCallback,
		writeCallback: writeCallback,
	}
}
