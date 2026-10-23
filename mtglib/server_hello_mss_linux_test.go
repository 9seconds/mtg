//go:build linux

package mtglib

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"io"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// mssTestPair opens a listener with TCP_MAXSEG = listenerMSS (0 leaves it
// alone) and returns the accepted server and the client connections.
// clientRcvBuf > 0 sets SO_RCVBUF of the client before connect (a small
// receive window).
func mssTestPair(t *testing.T, listenerMSS, clientRcvBuf int) (*net.TCPConn, *net.TCPConn) {
	t.Helper()

	// network.ListenControlMSS cannot be imported from here (import cycle);
	// it is tested in the network package, here the same is done directly.
	lc := net.ListenConfig{Control: func(_, _ string, conn syscall.RawConn) error {
		if listenerMSS <= 0 {
			return nil
		}

		var opErr error

		if err := conn.Control(func(fd uintptr) {
			opErr = unix.SetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG, listenerMSS)
		}); err != nil {
			return err
		}

		return opErr
	}}

	ln, err := lc.Listen(t.Context(), "tcp4", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { ln.Close() }) //nolint: errcheck

	dialer := net.Dialer{}
	if clientRcvBuf > 0 {
		dialer.Control = func(_, _ string, conn syscall.RawConn) error {
			var opErr error

			err := conn.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, clientRcvBuf)
			})
			if err != nil {
				return err
			}

			return opErr
		}
	}

	client, err := dialer.DialContext(t.Context(), "tcp4", ln.Addr().String())
	require.NoError(t, err)

	t.Cleanup(func() { client.Close() }) //nolint: errcheck

	server, err := ln.Accept()
	require.NoError(t, err)

	t.Cleanup(func() { server.Close() }) //nolint: errcheck

	return server.(*net.TCPConn), client.(*net.TCPConn) //nolint: forcetypeassert
}

func tcpInfoOf(t *testing.T, conn *net.TCPConn) *unix.TCPInfo {
	t.Helper()

	rawConn, err := conn.SyscallConn()
	require.NoError(t, err)

	info, err := getTCPInfo(rawConn)
	require.NoError(t, err)

	return info
}

// waitAllAcked waits until everything the server sent is acknowledged, so
// that tcpi_segs_out no longer changes.
func waitAllAcked(t *testing.T, conn *net.TCPConn) *unix.TCPInfo {
	t.Helper()

	var info *unix.TCPInfo

	require.Eventually(t, func() bool {
		info = tcpInfoOf(t, conn)

		return info.Notsent_bytes == 0 && info.Unacked == 0
	}, 5*time.Second, time.Millisecond)

	return info
}

func readAllAsync(conn net.Conn, size int) <-chan []byte {
	done := make(chan []byte, 1)

	go func() {
		buf := make([]byte, size)
		n, _ := io.ReadFull(conn, buf)
		done <- buf[:n]
	}()

	return done
}

// The main scenario: the socket is at bulk 1400, the ServerHello (~4 KB) is
// split by 92 - at least one segment per chunk, and the data after it goes in
// large segments (tcpi_snd_mss >= 1000).
func TestWriteFragmentedSmallHelloThenBulk(t *testing.T) {
	server, client := mssTestPair(t, 1400, 0)

	clientMSS, err := getsockoptMSS(client)
	require.NoError(t, err)
	assert.LessOrEqual(t, clientMSS, 1400)
	assert.GreaterOrEqual(t, clientMSS, 1000)

	hello := make([]byte, 4096)
	_, err = rand.Read(hello)
	require.NoError(t, err)

	got := readAllAsync(client, len(hello))

	before := waitAllAcked(t, server)
	chunk := segmentPayload(92, before)
	assert.LessOrEqual(t, chunk, 80+12)

	n, err := writeFragmented(server, hello, 92, time.Now().Add(5*time.Second))
	require.NoError(t, err)
	assert.Equal(t, len(hello), n)

	afterHello := waitAllAcked(t, server)
	helloSegs := int(afterHello.Segs_out - before.Segs_out)
	wantSegs := (len(hello) + chunk - 1) / chunk

	assert.GreaterOrEqual(t, helloSegs, wantSegs, "the ServerHello must leave in chunks of %d bytes", chunk)
	assert.Equal(t, hello, <-got)

	bulk := make([]byte, 64*1024)
	got = readAllAsync(client, len(bulk))

	_, err = server.Write(bulk)
	require.NoError(t, err)

	afterBulk := waitAllAcked(t, server)
	bulkSegs := int(afterBulk.Segs_out - afterHello.Segs_out)

	assert.GreaterOrEqual(t, int(afterBulk.Snd_mss), 1000)
	assert.Less(t, bulkSegs, len(bulk)/500, "segments after the ServerHello must be large")
	assert.Len(t, <-got, len(bulk))

	t.Logf("ServerHello %d bytes: chunk %d, segments %d (want >= %d); "+
		"bulk 64 KB: segments %d, tcpi_snd_mss before=%d after=%d",
		len(hello), chunk, helloSegs, wantSegs, bulkSegs, before.Snd_mss, afterBulk.Snd_mss)
}

// Chunks must not be merged when sending is window-limited: a client with a
// tiny receive window does not read for 300 ms, the kernel keeps the data
// queued. Without waiting for tcpi_notsent_bytes = 0 the next chunks would be
// appended to the unsent tail of the queue and leave as one large segment.
func TestWriteFragmentedDoesNotCoalesceWhenWindowLimited(t *testing.T) {
	server, client := mssTestPair(t, 1400, 2048)

	hello := make([]byte, 4096)
	_, err := rand.Read(hello)
	require.NoError(t, err)

	before := waitAllAcked(t, server)
	chunk := segmentPayload(92, before)

	got := make(chan []byte, 1)

	go func() {
		time.Sleep(300 * time.Millisecond)

		buf := make([]byte, len(hello))
		n, _ := io.ReadFull(client, buf)
		got <- buf[:n]
	}()

	_, err = writeFragmented(server, hello, 92, time.Now().Add(5*time.Second))
	require.NoError(t, err)

	assert.Equal(t, hello, <-got)

	after := waitAllAcked(t, server)
	segs := int(after.Segs_out - before.Segs_out)
	wantSegs := (len(hello) + chunk - 1) / chunk

	assert.GreaterOrEqual(t, segs, wantSegs)
	t.Logf("receive window 2 KB: ServerHello of %d bytes left in %d segments (%d chunks)", len(hello), segs, wantSegs)
}

// A real ServerHello written through fragmentedWriter is parsed by the client
// as three TLS records: splitting does not corrupt the stream.
func TestSendServerHelloThroughFragmentedWriter(t *testing.T) {
	server, client := mssTestPair(t, 1400, 0)

	hello := &fake.ClientHello{CipherSuite: 4867, SessionID: make([]byte, 32)}
	_, err := rand.Read(hello.SessionID)
	require.NoError(t, err)

	secret := GenerateSecret("example.com")
	writer := fragmentedWriter{conn: server, mss: 92, deadline: time.Now().Add(5 * time.Second)}

	before := waitAllAcked(t, server)

	errCh := make(chan error, 1)

	go func() {
		errCh <- fake.SendServerHello(writer, secret.Key[:], hello, fake.NoiseParams{})
	}()

	reader := bufio.NewReader(client)
	total := 0

	for _, want := range []byte{tls.TypeHandshake, tls.TypeChangeCipherSpec, tls.TypeApplicationData} {
		rec := &bytes.Buffer{}

		recordType, length, err := tls.ReadRecord(reader, rec)
		require.NoError(t, err)
		assert.Equal(t, want, recordType)

		total += 5 + int(length)
	}

	require.NoError(t, <-errCh)

	after := waitAllAcked(t, server)
	chunk := segmentPayload(92, before)

	assert.GreaterOrEqual(t, int(after.Segs_out-before.Segs_out), (total+chunk-1)/chunk)
}

// The PROXY protocol wrapper (TCPConn()) is TCP as well: splitting reaches it.
func TestTCPConnOfUnwrapsProxyProtocol(t *testing.T) {
	server, _ := mssTestPair(t, 0, 0)

	assert.Same(t, server, tcpConnOf(server))
	assert.Same(t, server, tcpConnOf(tcpWrapper{conn: server}))
	assert.Nil(t, tcpConnOf(nonTCPConn{}))
}

type tcpWrapper struct {
	net.Conn

	conn *net.TCPConn
}

func (w tcpWrapper) TCPConn() (*net.TCPConn, bool) { return w.conn, true }

type nonTCPConn struct {
	net.Conn
}

// Not TCP: the data is written in one go, without errors.
func TestWriteFragmentedFallsBackForNonTCP(t *testing.T) {
	left, right := net.Pipe()
	defer left.Close()  //nolint: errcheck
	defer right.Close() //nolint: errcheck

	got := readAllAsync(right, 3)

	n, err := writeFragmented(left, []byte("abc"), 92, time.Time{})
	require.NoError(t, err)
	assert.Equal(t, 3, n)
	assert.Equal(t, []byte("abc"), <-got)
}

func getsockoptMSS(conn *net.TCPConn) (int, error) {
	rawConn, err := conn.SyscallConn()
	if err != nil {
		return 0, err //nolint: wrapcheck
	}

	var (
		mss    int
		sysErr error
	)

	if err := rawConn.Control(func(fd uintptr) {
		mss, sysErr = unix.GetsockoptInt(int(fd), unix.IPPROTO_TCP, unix.TCP_MAXSEG)
	}); err != nil {
		return 0, err //nolint: wrapcheck
	}

	return mss, sysErr //nolint: wrapcheck
}
