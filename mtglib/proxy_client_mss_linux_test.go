//go:build linux

package mtglib_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/internal/proxyprotocol"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yl2chen/cidranger"
	"golang.org/x/sys/unix"
)

// The secret that signed the real ClientHello snapshots in testdata of the
// fake package.
const snapshotSecret = "ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d"

func snapshotClientHello(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(
		"internal", "tls", "fake", "testdata", "client-hello-ok-19dfe38384b9884b.json"))
	require.NoError(t, err)

	snapshot := struct {
		Full string `json:"full"`
	}{}
	require.NoError(t, json.Unmarshal(data, &snapshot))

	full, err := base64.StdEncoding.DecodeString(snapshot.Full)
	require.NoError(t, err)

	return full
}

// handshakeSetup tunes one test handshake through the proxy.
type handshakeSetup struct {
	serverHelloMSS      int
	serverHelloMSSBytes int
	handshakeTimeout    time.Duration

	// proxyProtocol > 0 puts the listener behind the PROXY protocol adapter
	// (as proxy-protocol-listener = true does) and sends a header of this
	// version before the ClientHello.
	proxyProtocol byte

	// clientRcvBuf > 0 sets SO_RCVBUF of the client before connect.
	clientRcvBuf int
}

// startHandshake starts a proxy with the given setup on a loopback listener,
// connects to it and sends the PROXY header (if any). It returns the client
// connection and a channel that reports the end of the stream on the proxy
// side. The ClientHello is sent by sendClientHello.
func startHandshake(t *testing.T, setup handshakeSetup) (*net.TCPConn, <-chan struct{}) {
	t.Helper()

	secret, err := mtglib.ParseSecret(snapshotSecret)
	require.NoError(t, err)

	dialer, err := network.NewDefaultDialer(0, 0)
	require.NoError(t, err)

	ntw, err := network.NewNetwork(dialer, "mtgtest", "1.1.1.1", 0)
	require.NoError(t, err)

	allowlist, err := ipblocklist.NewFireholFromFiles(logger.NewNoopLogger(), 1,
		[]files.File{files.NewMem([]*net.IPNet{cidranger.AllIPv4, cidranger.AllIPv6})}, nil)
	require.NoError(t, err)

	go allowlist.Run(time.Second)

	t.Cleanup(allowlist.Shutdown)

	require.Eventually(t, func() bool {
		return allowlist.Contains(net.ParseIP("127.0.0.1"))
	}, 2*time.Second, 10*time.Millisecond)

	finished := streamFinished(make(chan struct{}, 1))

	proxy, err := mtglib.NewProxy(mtglib.ProxyOpts{
		Secret:          secret,
		Network:         ntw,
		AntiReplayCache: antireplay.NewNoop(),
		IPBlocklist:     ipblocklist.NewNoop(),
		IPAllowlist:     allowlist,
		EventStream:     finished,
		Logger:          logger.NewNoopLogger(),
		UseTestDCs:      true,
		// The ClientHello snapshot was taken in 2021.
		TolerateTimeSkewness: 100 * 365 * 24 * time.Hour,
		HandshakeTimeout:     setup.handshakeTimeout,
		ServerHelloMSS:       setup.serverHelloMSS,
		ServerHelloMSSBytes:  setup.serverHelloMSSBytes,
	})
	require.NoError(t, err)

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)

	if tcpListener := listener; setup.proxyProtocol > 0 {
		listener = &proxyprotocol.ListenerAdapter{
			Listener: proxyproto.Listener{Listener: tcpListener},
		}
	}

	go proxy.Serve(listener) //nolint: errcheck

	clientDialer := net.Dialer{}
	if setup.clientRcvBuf > 0 {
		clientDialer.Control = func(_, _ string, conn syscall.RawConn) error {
			var opErr error

			if err := conn.Control(func(fd uintptr) {
				opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_RCVBUF, setup.clientRcvBuf)
			}); err != nil {
				return err
			}

			return opErr
		}
	}

	conn, err := clientDialer.DialContext(t.Context(), "tcp4", listener.Addr().String())
	require.NoError(t, err)

	done := make(chan struct{})

	go func() {
		defer close(done)

		select {
		case <-finished:
		case <-time.After(10 * time.Second):
		}
	}()

	// Let the stream finish before Shutdown: the proxy closes active streams
	// on Shutdown, which must not overlap with the handshake still going on.
	t.Cleanup(func() {
		conn.Close() //nolint: errcheck

		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("the stream has not finished")
		}

		listener.Close() //nolint: errcheck
		proxy.Shutdown()
	})

	client := conn.(*net.TCPConn) //nolint: forcetypeassert
	require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))

	if setup.proxyProtocol > 0 {
		header := proxyproto.HeaderProxyFromAddrs(setup.proxyProtocol,
			&net.TCPAddr{IP: net.ParseIP("10.1.2.3"), Port: 34567},
			client.RemoteAddr())

		_, err := header.WriteTo(client)
		require.NoError(t, err)
	}

	return client, done
}

func sendClientHello(t *testing.T, client net.Conn) {
	t.Helper()

	_, err := client.Write(snapshotClientHello(t))
	require.NoError(t, err)
}

// readServerHello reads the three TLS records of the FakeTLS server response
// and returns their total size.
func readServerHello(t *testing.T, client net.Conn) int {
	t.Helper()

	total := 0

	for range 3 { // ServerHello, ChangeCipherSpec, ApplicationData
		header := make([]byte, 5)
		_, err := io.ReadFull(client, header)
		require.NoError(t, err)

		body := make([]byte, binary.BigEndian.Uint16(header[3:]))
		_, err = io.ReadFull(client, body)
		require.NoError(t, err)

		total += len(header) + len(body)
	}

	return total
}

// serverHelloSegments runs a real FakeTLS handshake through the proxy and
// returns how many segments the client received while reading the
// ServerHello (tcpi_segs_in), the largest segment payload the client has seen
// (tcpi_rcv_mss, see below) and the ServerHello size.
func serverHelloSegments(t *testing.T, setup handshakeSetup) (int, int, int) {
	t.Helper()

	client, _ := startHandshake(t, setup)

	// Taken before the ClientHello: the server answers right away, and a slow
	// test goroutine (-race) could otherwise miss the first segments.
	before := clientTCPInfo(t, client)

	sendClientHello(t, client)

	// tcpi_segs_in also counts pure ACKs of the ClientHello, so it can only
	// overstate the number of data segments; the assertions below use it as a
	// lower bound for the shaped case and an upper bound for the plain one
	// with a margin.
	total := readServerHello(t, client)
	after := clientTCPInfo(t, client)

	return int(after.Segs_in - before.Segs_in), int(after.Rcv_mss), total
}

func clientTCPInfo(t *testing.T, conn *net.TCPConn) *unix.TCPInfo {
	t.Helper()

	rawConn, err := conn.SyscallConn()
	require.NoError(t, err)

	var (
		info   *unix.TCPInfo
		sysErr error
	)

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		info, sysErr = unix.GetsockoptTCPInfo(int(fd), unix.IPPROTO_TCP, unix.TCP_INFO)
	}))
	require.NoError(t, sysErr)

	return info
}

// tcpi_rcv_mss of the client is the receiver's estimate of the sender MSS:
// it starts at no more than 536 (TCP_MSS_DEFAULT) and is raised to the
// payload of any received segment that is at least as large. So it stays at
// or below 536 only if no segment above 536 bytes has arrived. This is the
// best signal of observed segment sizes available without CAP_NET_RAW; the
// exact per-segment check is TestProxyServerHelloMSSCapturedSegments.
const rcvMSSDefault = 536

// wholeResponse as ServerHelloMSSBytes splits the whole FakeTLS server
// response, not only the ServerHello record.
const wholeResponse = 1 << 16

// With ServerHelloMSS and a prefix above the response size the proxy sends
// the whole server response in small segments (80 bytes of payload here),
// without it - in a few large ones.
func TestProxyServerHelloMSS(t *testing.T) {
	plainSegs, plainRcvMSS, plainSize := serverHelloSegments(t, handshakeSetup{})
	shapedSegs, shapedRcvMSS, shapedSize := serverHelloSegments(t, handshakeSetup{serverHelloMSS: 92, serverHelloMSSBytes: wholeResponse})

	assert.LessOrEqual(t, plainSegs, plainSize/1000+3)
	assert.Greater(t, plainRcvMSS, rcvMSSDefault)

	assert.GreaterOrEqual(t, shapedSegs, shapedSize/80)
	assert.LessOrEqual(t, shapedRcvMSS, rcvMSSDefault)

	t.Logf("ServerHello without client-mss: %d bytes, %d segments, tcpi_rcv_mss %d; "+
		"with client-mss 92: %d bytes, %d segments, tcpi_rcv_mss %d",
		plainSize, plainSegs, plainRcvMSS, shapedSize, shapedSegs, shapedRcvMSS)
}

// By default only the ServerHello record (about 130 bytes) is split: a couple
// of small segments, then the rest of the response in large ones.
func TestProxyServerHelloMSSDefaultSplitsOnlyServerHello(t *testing.T) {
	segs, rcvMSS, size := serverHelloSegments(t, handshakeSetup{serverHelloMSS: 92})

	assert.Less(t, segs, size/80/4, "the whole response is split")
	assert.Greater(t, rcvMSS, rcvMSSDefault, "no large segment after the ServerHello")

	t.Logf("client-mss 92, default prefix: %d bytes, %d segments, tcpi_rcv_mss %d", size, segs, rcvMSS)
}

// Behind the PROXY protocol listener (proxy-protocol-listener = true) the
// accepted connection is a wrapper, not *net.TCPConn. The ServerHello must
// still be split, for both header versions, and the handshake must succeed.
func TestProxyServerHelloMSSBehindProxyProtocol(t *testing.T) {
	for _, version := range []byte{1, 2} {
		t.Run(fmt.Sprintf("v%d", version), func(t *testing.T) {
			segs, rcvMSS, size := serverHelloSegments(t, handshakeSetup{
				serverHelloMSS:      92,
				serverHelloMSSBytes: wholeResponse,
				proxyProtocol:       version,
			})

			assert.GreaterOrEqual(t, segs, size/80)
			assert.LessOrEqual(t, rcvMSS, rcvMSSDefault)
			t.Logf("PROXY v%d: ServerHello %d bytes, %d segments, tcpi_rcv_mss %d", version, size, segs, rcvMSS)
		})
	}
}

// A client that sends the ClientHello and never reads, with a receive window
// smaller than the ServerHello: the small-segment writer has to wait for the
// window, and that wait must end at the handshake deadline, not hang the
// stream.
func TestProxyServerHelloMSSNonReadingClientRespectsDeadline(t *testing.T) {
	const handshakeTimeout = 500 * time.Millisecond

	started := time.Now()

	client, finished := startHandshake(t, handshakeSetup{
		serverHelloMSS:      92,
		serverHelloMSSBytes: wholeResponse,
		handshakeTimeout:    handshakeTimeout,
		clientRcvBuf:        1024, // the kernel rounds it up to its minimum
	})
	sendClientHello(t, client)

	select {
	case <-finished:
	case <-time.After(handshakeTimeout + 3*time.Second):
		t.Fatal("the stream is still waiting for a client that does not read")
	}

	elapsed := time.Since(started)
	assert.GreaterOrEqual(t, elapsed, handshakeTimeout/2)

	// The window really was the limit: what the client has in its receive
	// queue is not the whole server response (otherwise this test would not
	// exercise the wait at all).
	queued := peekQueued(t, client)
	assert.False(t, hasFullServerHello(queued), "the whole ServerHello fit into the window")

	t.Logf("non-reading client: stream finished after %v, %d bytes in the client receive queue",
		elapsed, len(queued))
}

// peekQueued returns the unread data in the receive queue of conn without
// consuming it.
func peekQueued(t *testing.T, conn *net.TCPConn) []byte {
	t.Helper()

	rawConn, err := conn.SyscallConn()
	require.NoError(t, err)

	buf := make([]byte, 64*1024)

	var (
		n      int
		sysErr error
	)

	require.NoError(t, rawConn.Control(func(fd uintptr) {
		n, _, sysErr = unix.Recvfrom(int(fd), buf, unix.MSG_PEEK|unix.MSG_DONTWAIT)
	}))

	if errors.Is(sysErr, unix.EAGAIN) {
		return nil
	}

	require.NoError(t, sysErr)

	return buf[:n]
}

// hasFullServerHello tells whether data holds the three complete TLS records
// of the FakeTLS server response.
func hasFullServerHello(data []byte) bool {
	for range 3 {
		if len(data) < 5 {
			return false
		}

		length := 5 + int(binary.BigEndian.Uint16(data[3:5]))
		if len(data) < length {
			return false
		}

		data = data[length:]
	}

	return true
}

// The exact check: every segment the proxy sends while writing the
// ServerHello carries payload + TCP options of at most client-mss bytes. It
// captures loopback packets with an AF_PACKET socket, which needs CAP_NET_RAW
// (root in a container has it, an unprivileged CI user does not); without it
// the test is skipped and TestProxyServerHelloMSS remains the signal.
func TestProxyServerHelloMSSCapturedSegments(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup handshakeSetup
	}{
		{name: "direct", setup: handshakeSetup{serverHelloMSS: 92, serverHelloMSSBytes: wholeResponse}},
		{name: "PROXY v2", setup: handshakeSetup{serverHelloMSS: 92, serverHelloMSSBytes: wholeResponse, proxyProtocol: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := startLoopbackCapture(t)

			client, _ := startHandshake(t, tc.setup)
			sendClientHello(t, client)
			size := readServerHello(t, client)
			segments := capture.serverSegments(t, client.RemoteAddr().(*net.TCPAddr).Port) //nolint: forcetypeassert

			payload, largest := 0, 0

			for _, seg := range segments {
				payload += seg.payload
				largest = max(largest, seg.payload+seg.options)

				assert.LessOrEqual(t, seg.payload+seg.options, 92, "segment of %d bytes + %d of options",
					seg.payload, seg.options)
			}

			assert.GreaterOrEqual(t, payload, size, "not all ServerHello segments were captured")
			t.Logf("%s: ServerHello %d bytes in %d captured segments, the largest is %d bytes with options",
				tc.name, size, len(segments), largest)
		})
	}

	// Sanity check of the capture itself: without client-mss the same
	// ServerHello leaves in segments far above 92 bytes.
	capture := startLoopbackCapture(t)

	client, _ := startHandshake(t, handshakeSetup{})
	sendClientHello(t, client)
	readServerHello(t, client)

	largest := 0
	for _, seg := range capture.serverSegments(t, client.RemoteAddr().(*net.TCPAddr).Port) { //nolint: forcetypeassert
		largest = max(largest, seg.payload)
	}

	assert.Greater(t, largest, 1000)
}

// streamFinished is an EventStream that reports the end of a stream.
type streamFinished chan struct{}

func (s streamFinished) Send(_ context.Context, evt mtglib.Event) {
	if _, ok := evt.(mtglib.EventFinish); ok {
		select {
		case s <- struct{}{}:
		default:
		}
	}
}

// The exact check of the prefix mode: segments that carry the first prefix
// bytes of the response are at most client-mss with options and end exactly
// at the prefix, the rest of the response goes in large segments.
func TestProxyServerHelloMSSPrefixCapturedSegments(t *testing.T) {
	for _, tc := range []struct {
		name   string
		prefix int // 0 is the ServerHello record
	}{
		{name: "ServerHello record", prefix: 0},
		{name: "first 500 bytes", prefix: 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := startLoopbackCapture(t)

			client, _ := startHandshake(t, handshakeSetup{serverHelloMSS: 92, serverHelloMSSBytes: tc.prefix})
			sendClientHello(t, client)
			helloRecord, size := readServerHelloRecords(t, client)
			segments := capture.serverSegments(t, client.RemoteAddr().(*net.TCPAddr).Port) //nolint: forcetypeassert

			prefix := tc.prefix
			if prefix == 0 {
				prefix = helloRecord
			}

			offset, small, largestRest := 0, 0, 0

			for _, seg := range segments {
				if offset < prefix {
					small++

					assert.LessOrEqual(t, seg.payload+seg.options, 92, "segment at %d: %d bytes + %d of options",
						offset, seg.payload, seg.options)
					assert.LessOrEqual(t, offset+seg.payload, prefix, "segment at %d crosses the prefix", offset)
				} else {
					largestRest = max(largestRest, seg.payload)
				}

				offset += seg.payload
			}

			assert.GreaterOrEqual(t, offset, size, "not all segments were captured")
			assert.Greater(t, largestRest, 1000, "the rest of the response must go in large segments")

			t.Logf("%s: %d bytes split in %d segments, the rest %d bytes, its largest segment %d",
				tc.name, prefix, small, size-prefix, largestRest)
		})
	}
}

// readServerHelloRecords reads the FakeTLS server response and returns the
// size of the ServerHello record and of the whole response.
func readServerHelloRecords(t *testing.T, client net.Conn) (int, int) {
	t.Helper()

	sizes := []int{}

	for range 3 {
		header := make([]byte, 5)
		_, err := io.ReadFull(client, header)
		require.NoError(t, err)

		body := make([]byte, binary.BigEndian.Uint16(header[3:]))
		_, err = io.ReadFull(client, body)
		require.NoError(t, err)

		sizes = append(sizes, len(header)+len(body))
	}

	return sizes[0], sizes[0] + sizes[1] + sizes[2]
}
