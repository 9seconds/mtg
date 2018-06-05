package wrappers

import "io"

// TrafficReadWriteCloser counts an amount of ingress/egress traffic by
// calling given callbacks.
type TrafficReadWriteCloser struct {
	conn          io.ReadWriteCloser
	readCallback  func(int)
	writeCallback func(int)
}

// Read reads from connection
func (t *TrafficReadWriteCloser) Read(p []byte) (n int, err error) {
	n, err = t.conn.Read(p)
	t.readCallback(n)
	return
}

// Write writes into connection.
func (t *TrafficReadWriteCloser) Write(p []byte) (n int, err error) {
	n, err = t.conn.Write(p)
	t.writeCallback(n)
	return
}

// Close closes underlying connection.
func (t *TrafficReadWriteCloser) Close() error {
	return t.conn.Close()
}

func NewTrafficRWC(conn io.ReadWriteCloser, readCallback, writeCallback func(int)) io.ReadWriteCloser {
	return &TrafficReadWriteCloser{
		conn:          conn,
		readCallback:  readCallback,
		writeCallback: writeCallback,
	}
}
