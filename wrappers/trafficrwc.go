package wrappers

import "net"

// TrafficReadWriteCloser counts an amount of ingress/egress traffic by
// calling given callbacks.
type TrafficReadWriteCloserWithAddr struct {
	conn          ReadWriteCloserWithAddr
	readCallback  func(int)
	writeCallback func(int)
}

// Read reads from connection
func (t *TrafficReadWriteCloserWithAddr) Read(p []byte) (n int, err error) {
	n, err = t.conn.Read(p)
	t.readCallback(n)
	return
}

// Write writes into connection.
func (t *TrafficReadWriteCloserWithAddr) Write(p []byte) (n int, err error) {
	n, err = t.conn.Write(p)
	t.writeCallback(n)
	return
}

// Close closes underlying connection.
func (t *TrafficReadWriteCloserWithAddr) Close() error {
	return t.conn.Close()
}

func (t *TrafficReadWriteCloserWithAddr) Addr() *net.TCPAddr {
	return t.conn.Addr()
}

// NewTrafficRWC wraps ReadWriteCloser to have read/write callbacks.
func NewTrafficRWC(conn ReadWriteCloserWithAddr, readCallback, writeCallback func(int)) ReadWriteCloserWithAddr {
	return &TrafficReadWriteCloserWithAddr{
		conn:          conn,
		readCallback:  readCallback,
		writeCallback: writeCallback,
	}
}
