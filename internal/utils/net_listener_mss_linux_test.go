//go:build linux

package utils_test

import (
	"net"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/proxyprotocol"
	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func tcpMaxSeg(t *testing.T, conn *net.TCPConn) int {
	t.Helper()

	rawConn, err := conn.SyscallConn()
	require.NoError(t, err)

	var (
		mss    int
		sysErr error
	)

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		mss, sysErr = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG)
	}))
	require.NoError(t, sysErr)

	return mss
}

// mssListener returns a TCP listener with TCP_MAXSEG = mss set through
// SetListenerMSS on the utils.Listener wrapper (as run_proxy does after
// NewListener); connections are accepted from the TCP listener itself.
func mssListener(t *testing.T, mss int) net.Listener {
	t.Helper()

	base, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { base.Close() }) //nolint: errcheck

	require.NoError(t, utils.SetListenerMSS(&utils.Listener{Listener: base}, mss))

	return base
}

func dial(t *testing.T, addr net.Addr) *net.TCPConn {
	t.Helper()

	conn, err := net.DialTimeout("tcp", addr.String(), 2*time.Second)
	require.NoError(t, err)

	t.Cleanup(func() { conn.Close() }) //nolint: errcheck

	return conn.(*net.TCPConn) //nolint: forcetypeassert
}

// TCP_MAXSEG set on an already listening socket is inherited by connections
// accepted after that and advertised to the client in the SYN-ACK, just as
// if it were set before listen().
func TestSetListenerMSSAppliesToAcceptedConnections(t *testing.T) {
	listener := mssListener(t, 92)
	client := dial(t, listener.Addr())

	conn, err := listener.Accept()
	require.NoError(t, err)

	defer conn.Close() //nolint: errcheck

	assert.LessOrEqual(t, tcpMaxSeg(t, client), 92)
	assert.LessOrEqual(t, tcpMaxSeg(t, conn.(*net.TCPConn)), 92) //nolint: forcetypeassert
}

func TestSetListenerMSSRejectsTooSmall(t *testing.T) {
	base, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer base.Close() //nolint: errcheck

	assert.Error(t, utils.SetListenerMSS(&utils.Listener{Listener: base}, 40))
}

// NewListener may return the wrapper by value; SetListenerMSS reaches the
// socket under it as well. Written without a static conversion, so that the
// test still builds if Listener implements net.Listener by pointer only.
func TestSetListenerMSSUnwrapsListenerValue(t *testing.T) {
	base, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer base.Close() //nolint: errcheck

	listener, ok := any(utils.Listener{Listener: base}).(net.Listener)
	if !ok {
		t.Skip("utils.Listener implements net.Listener by pointer only")
	}

	require.NoError(t, utils.SetListenerMSS(listener, 92))

	client := dial(t, base.Addr())

	conn, err := base.Accept()
	require.NoError(t, err)

	defer conn.Close() //nolint: errcheck

	assert.LessOrEqual(t, tcpMaxSeg(t, client), 92)
}

// proxy-protocol-listener wraps the listener after SetListenerMSS: the
// accepted connections are PROXY protocol wrappers, the socket under them
// still has the listener MSS.
func TestSetListenerMSSBehindProxyProtocol(t *testing.T) {
	for _, version := range []byte{1, 2} {
		listener := mssListener(t, 92)
		wrapped := &proxyprotocol.ListenerAdapter{
			Listener: proxyproto.Listener{Listener: listener},
		}

		client := dial(t, listener.Addr())

		header := proxyproto.HeaderProxyFromAddrs(version,
			&net.TCPAddr{IP: net.ParseIP("10.1.2.3"), Port: 34567}, listener.Addr())
		_, err := header.WriteTo(client)
		require.NoError(t, err)

		conn, err := wrapped.Accept()
		require.NoError(t, err)

		// The header is parsed lazily, on the first read or address lookup.
		assert.Equal(t, "10.1.2.3:34567", conn.RemoteAddr().String())

		tcp, ok := conn.(interface{ TCPConn() (*net.TCPConn, bool) }).TCPConn() //nolint: forcetypeassert
		require.True(t, ok)

		assert.LessOrEqual(t, tcpMaxSeg(t, tcp), 92, "PROXY v%d", version)
		assert.LessOrEqual(t, tcpMaxSeg(t, client), 92, "PROXY v%d", version)

		conn.Close() //nolint: errcheck
	}
}
