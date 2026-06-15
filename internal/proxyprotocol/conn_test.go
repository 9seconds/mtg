package proxyprotocol

import (
	"net"
	"syscall"
	"testing"

	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/require"
)

func TestConnWrapperSyscallConn(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close() //nolint: errcheck

	accepted := make(chan net.Conn, 1)
	acceptErr := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			acceptErr <- err

			return
		}
		accepted <- conn
	}()

	clientConn, err := net.Dial("tcp4", listener.Addr().String())
	require.NoError(t, err)
	defer clientConn.Close() //nolint: errcheck

	var conn net.Conn
	select {
	case err := <-acceptErr:
		require.NoError(t, err)
	case conn = <-accepted:
	}
	defer conn.Close() //nolint: errcheck

	wrapped := connWrapper{proxyproto.NewConn(conn)}
	_, ok := any(wrapped).(syscall.Conn)
	require.True(t, ok)

	rawConn, err := wrapped.SyscallConn()
	require.NoError(t, err)
	require.NotNil(t, rawConn)
}
