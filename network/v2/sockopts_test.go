//go:build linux || darwin
// +build linux darwin

package network

import (
	"net"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func tcpKeepIdleOption() int {
	if runtime.GOOS == "darwin" {
		return 0x10 // TCP_KEEPALIVE on macOS
	}

	return 0x4 // TCP_KEEPIDLE on Linux
}

func TestSetCommonSocketOptionsKeepAlive(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer func() {
		err := listener.Close()
		require.NoError(t, err)
	}()

	type dialResult struct {
		conn net.Conn
		err  error
	}

	dialDone := make(chan dialResult, 1)

	go func() {
		c, err := net.Dial("tcp", listener.Addr().String())
		dialDone <- dialResult{conn: c, err: err}
	}()

	tcpListener, ok := listener.(*net.TCPListener)
	require.True(t, ok, "listener must be a *net.TCPListener")

	require.NoError(t, tcpListener.SetDeadline(time.Now().Add(5*time.Second)))

	accepted, err := listener.Accept()
	require.NoError(t, err)
	defer func() {
		err := accepted.Close()
		require.NoError(t, err)
	}()

	dr := <-dialDone
	require.NoError(t, dr.err)
	defer func() {
		err := dr.conn.Close()
		require.NoError(t, err)
	}()

	tcpConn := accepted.(*net.TCPConn)

	err = setCommonSocketOptions(tcpConn, DefaultKeepAliveConfig, DefaultTCPNotSentLowat)
	require.NoError(t, err)

	rawConn, err := tcpConn.SyscallConn()
	require.NoError(t, err)

	err = rawConn.Control(func(fd uintptr) {
		val, err := unix.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE)
		require.NoError(t, err)
		require.NotEqual(t, 0, val, "SO_KEEPALIVE should be enabled")

		idle, err := unix.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, tcpKeepIdleOption())
		require.NoError(t, err)
		require.Equal(t, 15, idle, "keepalive idle should match DefaultKeepAliveIdle")

		interval, err := unix.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, unix.TCP_KEEPINTVL)
		require.NoError(t, err)
		require.Equal(t, 15, interval, "keepalive interval should match DefaultKeepAliveInterval")

		count, err := unix.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, unix.TCP_KEEPCNT)
		require.NoError(t, err)
		require.Equal(t, 9, count, "keepalive count should match DefaultKeepAliveCount")
	})
	require.NoError(t, err)
}

func TestSetCommonSocketOptionsNotSentLowat(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		want int
	}{
		{name: "default", want: DefaultTCPNotSentLowat},
		{name: "custom", want: 4 * 1024 * 1024},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			listener, err := net.Listen("tcp", "127.0.0.1:0")
			require.NoError(t, err)
			defer listener.Close() //nolint: errcheck

			dialDone := make(chan struct{})

			go func() {
				c, err := net.Dial("tcp", listener.Addr().String())
				if err == nil {
					defer c.Close() //nolint: errcheck
				}
				close(dialDone)
			}()

			require.NoError(t, listener.(*net.TCPListener).SetDeadline(time.Now().Add(5*time.Second)))

			accepted, err := listener.Accept()
			require.NoError(t, err)
			defer accepted.Close() //nolint: errcheck

			<-dialDone

			tcpConn := accepted.(*net.TCPConn)

			require.NoError(t, setCommonSocketOptions(tcpConn, DefaultKeepAliveConfig, tc.want))

			rawConn, err := tcpConn.SyscallConn()
			require.NoError(t, err)

			err = rawConn.Control(func(fd uintptr) {
				got, err := unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_NOTSENT_LOWAT)
				require.NoError(t, err)
				require.Equal(t, tc.want, got, "TCP_NOTSENT_LOWAT should match value passed to setCommonSocketOptions")
			})
			require.NoError(t, err)
		})
	}
}
