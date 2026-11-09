//go:build linux

package mtglib

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"os"
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
// queued. Without MSG_EOR the next chunks would be appended to the unsent tail
// of the queue and leave as one large segment.
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

// A client that never reads, with a receive window much smaller than the
// data: once the window is full, waiting for the queue to drain must stop at
// the deadline with os.ErrDeadlineExceeded instead of hanging.
func TestWriteFragmentedRespectsDeadlineWhenClientDoesNotRead(t *testing.T) {
	server, _ := mssTestPair(t, 1400, 2048)

	data := make([]byte, 64*1024)
	deadline := time.Now().Add(300 * time.Millisecond)
	started := time.Now()

	n, err := writeFragmented(server, data, 92, deadline)
	elapsed := time.Since(started)

	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	assert.Less(t, n, len(data))
	assert.GreaterOrEqual(t, elapsed, 250*time.Millisecond)
	assert.Less(t, elapsed, time.Second, "the write must return at the deadline")

	t.Logf("non-reading client: %d of %d bytes written, returned after %v", n, len(data), elapsed)
}

func setSndBuf(t *testing.T, conn *net.TCPConn, size int) {
	t.Helper()

	rawConn, err := conn.SyscallConn()
	require.NoError(t, err)

	var opErr error

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_SNDBUF, size)
	}))
	require.NoError(t, opErr)
}

// A full send buffer: the client does not read for a while, the server's send
// buffer is small, so sendmsg returns EAGAIN and partial sends in the middle
// of a chunk. The write waits for the socket to become writable, the data
// arrives intact, and every chunk still leaves in its own segment: the rest
// of a partially sent chunk joins it, not the next chunk.
func TestWriteFragmentedSendBufferFull(t *testing.T) {
	server, client := mssTestPair(t, 1400, 2048)
	setSndBuf(t, server, 4096)

	hello := make([]byte, 16*1024)
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

	n, err := writeFragmented(server, hello, 92, time.Now().Add(5*time.Second))
	require.NoError(t, err)
	assert.Equal(t, len(hello), n)
	assert.Equal(t, hello, <-got)

	after := waitAllAcked(t, server)
	segs := int(after.Segs_out - before.Segs_out)
	wantSegs := (len(hello) + chunk - 1) / chunk

	assert.GreaterOrEqual(t, segs, wantSegs)
	t.Logf("send buffer 4 KB, receive window 2 KB: %d bytes left in %d segments (%d chunks)",
		len(hello), segs, wantSegs)
}

// With a zero deadline the write deadline the connection already has applies:
// a client that never reads cannot hold the write past it.
func TestWriteFragmentedZeroDeadlineKeepsConnDeadline(t *testing.T) {
	server, _ := mssTestPair(t, 1400, 2048)
	setSndBuf(t, server, 4096)

	require.NoError(t, server.SetWriteDeadline(time.Now().Add(300*time.Millisecond)))

	started := time.Now()
	_, err := writeFragmented(server, make([]byte, 1<<20), 92, time.Time{})

	require.ErrorIs(t, err, os.ErrDeadlineExceeded)
	assert.Less(t, time.Since(started), 3*time.Second)
}

// sendmsgCall is one call of a fake sendmsg.
type sendmsgCall struct {
	data  []byte
	flags int
}

// fakeSendmsg records every call and answers it with the next result; once
// the results run out, it sends everything it is given.
type fakeSendmsg struct {
	calls   []sendmsgCall
	results []fakeSendmsgResult

	// wake makes the socket report readiness again after a fake EAGAIN (see
	// fakeSendmsgConn).
	wake func()
}

type fakeSendmsgResult struct {
	n   int
	err error
}

func (f *fakeSendmsg) sendmsg(_ int, p, _ []byte, _ unix.Sockaddr, flags int) (int, error) {
	f.calls = append(f.calls, sendmsgCall{data: bytes.Clone(p), flags: flags})

	if len(f.results) == 0 {
		return len(p), nil
	}

	res := f.results[0]
	f.results = f.results[1:]

	if errors.Is(res.err, unix.EAGAIN) {
		f.wake()
	}

	if res.err != nil {
		return -1, res.err
	}

	return res.n, nil
}

// fakeSendmsgConn returns a fake sendmsg on top of a real server socket.
//
// After a fake EAGAIN the runtime poller waits for a new readiness event
// (edge-triggered epoll), which an idle writable socket never produces; a real
// EAGAIN sets SOCK_NOSPACE and the kernel wakes the socket once there is room.
// So on a fake EAGAIN the client sends a byte: the wakeup reports the socket
// as writable as well. The write deadline turns a missed wakeup into a test
// failure instead of a hang.
func fakeSendmsgConn(t *testing.T, results []fakeSendmsgResult) (syscall.RawConn, *fakeSendmsg) {
	t.Helper()

	server, client := mssTestPair(t, 0, 0)
	require.NoError(t, server.SetWriteDeadline(time.Now().Add(5*time.Second)))

	rawConn, err := server.SyscallConn()
	require.NoError(t, err)

	return rawConn, &fakeSendmsg{
		results: results,
		wake: func() {
			client.Write([]byte{0}) //nolint: errcheck
		},
	}
}

func patternData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 251)
	}

	return data
}

// Every chunk is one sendmsg call with MSG_EOR: this is what keeps the kernel
// from merging chunks in the queue and from collapsing them on retransmission
// (see TestWriteFragmentedRetransmissionIsNotCollapsed for the kernel side).
func TestSendChunksMarksEveryChunkEOR(t *testing.T) {
	data := patternData(250)
	rawConn, fake := fakeSendmsgConn(t, nil)

	n, err := sendChunks(rawConn, data, 80, fake.sendmsg)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	want := [][]byte{data[0:80], data[80:160], data[160:240], data[240:250]}
	require.Len(t, fake.calls, len(want))

	for i, call := range fake.calls {
		assert.Equal(t, want[i], call.data, "call %d", i)
		assert.NotZero(t, call.flags&unix.MSG_EOR, "call %d has no MSG_EOR", i)
		assert.NotZero(t, call.flags&unix.MSG_NOSIGNAL, "call %d has no MSG_NOSIGNAL", i)
	}
}

// A partial send, EINTR and EAGAIN in the middle of a chunk: the rest of the
// same chunk is sent next (boundaries are counted from the start of data, not
// from the last write), EAGAIN waits for the socket to become writable, and
// the data is sent exactly once.
func TestSendChunksPartialSendAndEAGAIN(t *testing.T) {
	data := patternData(200)
	rawConn, fake := fakeSendmsgConn(t, []fakeSendmsgResult{
		{n: 30},
		{err: unix.EINTR},
		{err: unix.EAGAIN},
		{n: 20},
		{n: 30},
	})

	n, err := sendChunks(rawConn, data, 80, fake.sendmsg)
	require.NoError(t, err)
	assert.Equal(t, len(data), n)

	want := [][]byte{
		data[0:80],    // 30 sent
		data[30:80],   // EINTR
		data[30:80],   // EAGAIN
		data[30:80],   // 20 sent
		data[50:80],   // 30 sent, the end of the first chunk
		data[80:160],  // the second chunk, whole
		data[160:200], // the tail
	}
	require.Len(t, fake.calls, len(want))

	for i, call := range fake.calls {
		assert.Equal(t, want[i], call.data, "call %d", i)
		assert.NotZero(t, call.flags&unix.MSG_EOR, "call %d has no MSG_EOR", i)
	}
}

// A send error stops the write and reports how much was sent before it.
func TestSendChunksReportsSendError(t *testing.T) {
	rawConn, fake := fakeSendmsgConn(t, []fakeSendmsgResult{{n: 80}, {err: unix.EPIPE}})

	n, err := sendChunks(rawConn, patternData(200), 80, fake.sendmsg)

	require.ErrorIs(t, err, unix.EPIPE)
	assert.Equal(t, 80, n)
	assert.Len(t, fake.calls, 2)
}

// setDropAll attaches (or detaches) a socket filter that drops every packet
// arriving at conn, so that conn does not acknowledge anything.
func setDropAll(t *testing.T, conn *net.TCPConn, drop bool) {
	t.Helper()

	rawConn, err := conn.SyscallConn()
	require.NoError(t, err)

	var opErr error

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		if !drop {
			opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_DETACH_FILTER, 0)

			return
		}

		filter := []unix.SockFilter{{Code: unix.BPF_RET | unix.BPF_K, K: 0}}
		opErr = unix.SetsockoptSockFprog(int(fd), unix.SOL_SOCKET, unix.SO_ATTACH_FILTER, &unix.SockFprog{
			Len:    uint16(len(filter)),
			Filter: &filter[0],
		})
	}))

	// Some CI sandboxes do not allow socket filters for unprivileged
	// processes; the test needs one to stop acknowledgements.
	if drop && (errors.Is(opErr, unix.EPERM) || errors.Is(opErr, unix.EACCES)) {
		t.Skipf("attaching a socket filter is not permitted here: %v", opErr)
	}

	require.NoError(t, opErr)
}

// A lost ServerHello is retransmitted in the same small segments. The client
// drops everything it receives, so nothing is acknowledged and the server
// retransmits on RTO. Without MSG_EOR the kernel collapses the small skbs of
// the retransmission queue into one full segment (tcp_retrans_collapse, on by
// default): on a lossy path the ServerHello would go out in large segments
// exactly when splitting matters.
func TestWriteFragmentedRetransmissionIsNotCollapsed(t *testing.T) {
	server, client := mssTestPair(t, 1400, 0)

	before := waitAllAcked(t, server)
	chunk := segmentPayload(92, before)

	// Fewer chunks than the initial congestion window, so that all of them
	// are sent at once and wait in the retransmission queue together.
	hello := make([]byte, 8*chunk)
	_, err := rand.Read(hello)
	require.NoError(t, err)

	setDropAll(t, client, true)

	_, err = writeFragmented(server, hello, 92, time.Now().Add(5*time.Second))
	require.NoError(t, err)

	var lost *unix.TCPInfo

	require.Eventually(t, func() bool {
		lost = tcpInfoOf(t, server)

		return lost.Retransmits > 0
	}, 10*time.Second, time.Millisecond, "no retransmission timeout")

	got := readAllAsync(client, len(hello))

	setDropAll(t, client, false)

	assert.Equal(t, hello, <-got)

	after := waitAllAcked(t, server)
	retransSegs := int(after.Total_retrans - before.Total_retrans)
	retransBytes := int(after.Bytes_retrans - before.Bytes_retrans)

	require.Positive(t, retransSegs)
	assert.LessOrEqual(t, retransBytes, retransSegs*chunk,
		"retransmitted segments must stay at %d bytes of payload", chunk)

	t.Logf("ServerHello of %d bytes in chunks of %d: %d bytes retransmitted in %d segments",
		len(hello), chunk, retransBytes, retransSegs)
}
