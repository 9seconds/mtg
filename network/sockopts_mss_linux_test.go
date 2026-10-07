//go:build linux

package network_test

import (
	"io"
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func tcpMaxSeg(t *testing.T, conn net.Conn) int {
	t.Helper()

	rawConn, err := conn.(*net.TCPConn).SyscallConn() //nolint: forcetypeassert
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

func dialPair(t *testing.T, listenerMSS int) (net.Conn, net.Conn) {
	t.Helper()

	lc := net.ListenConfig{Control: network.ListenControlMSS(listenerMSS)}

	ln, err := lc.Listen(t.Context(), "tcp4", "127.0.0.1:0")
	require.NoError(t, err)

	defer ln.Close() //nolint: errcheck

	client, err := net.Dial("tcp4", ln.Addr().String())
	require.NoError(t, err)

	t.Cleanup(func() { client.Close() }) //nolint: errcheck

	server, err := ln.Accept()
	require.NoError(t, err)

	t.Cleanup(func() { server.Close() }) //nolint: errcheck

	return server, client
}

// TCP_MAXSEG of the listening socket is advertised to the client in the
// SYN-ACK: both sides then send segments no larger than that
// (client-mss-bulk = 0).
func TestListenControlMSSAdvertisedInSynAck(t *testing.T) {
	server, client := dialPair(t, 92)

	clientMSS := tcpMaxSeg(t, client)
	serverMSS := tcpMaxSeg(t, server)

	assert.LessOrEqual(t, clientMSS, 92)
	assert.LessOrEqual(t, serverMSS, 92)
	t.Logf("listener MSS 92: client TCP_MAXSEG=%d, server TCP_MAXSEG=%d", clientMSS, serverMSS)
}

func TestListenControlMSSZeroKeepsDefault(t *testing.T) {
	server, client := dialPair(t, 0)

	assert.Greater(t, tcpMaxSeg(t, client), 1000)
	assert.Greater(t, tcpMaxSeg(t, server), 1000)
}

func TestListenControlMSSRejectsTooSmall(t *testing.T) {
	lc := net.ListenConfig{Control: network.ListenControlMSS(40)}

	_, err := lc.Listen(t.Context(), "tcp4", "127.0.0.1:0")
	assert.Error(t, err)
}

// This is why client-mss-bulk goes on the listener: setsockopt(TCP_MAXSEG) on
// an established connection does not raise the segment size, mss_clamp is
// fixed during the TCP handshake.
func TestTCPMaxSegCannotBeRaisedAfterAccept(t *testing.T) {
	server, client := dialPair(t, 92)

	rawConn, err := server.(*net.TCPConn).SyscallConn() //nolint: forcetypeassert
	require.NoError(t, err)

	var sysErr error

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		sysErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG, 1400)
	}))
	require.NoError(t, sysErr)

	done := make(chan struct{})

	go func() {
		defer close(done)

		buf := make([]byte, 64*1024)
		io.ReadFull(client, buf) //nolint: errcheck
	}()

	_, err = server.Write(make([]byte, 64*1024))
	require.NoError(t, err)
	<-done

	var info *unix.TCPInfo

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		info, sysErr = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
	}))
	require.NoError(t, sysErr)

	assert.LessOrEqual(t, int(info.Snd_mss), 92)
	t.Logf("listener MSS 92, then setsockopt 1400: tcpi_snd_mss=%d", info.Snd_mss)
}
