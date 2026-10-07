//go:build linux

package mtglib_test

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
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

// serverHelloSegments runs a real FakeTLS handshake through the proxy with
// the given ServerHelloMSS and returns how many segments the client received
// while reading the ServerHello (tcpi_segs_in), and the ServerHello size.
func serverHelloSegments(t *testing.T, serverHelloMSS int) (int, int) {
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
		ServerHelloMSS:       serverHelloMSS,
	})
	require.NoError(t, err)

	defer proxy.Shutdown()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)

	defer listener.Close() //nolint: errcheck

	go proxy.Serve(listener) //nolint: errcheck

	conn, err := net.Dial("tcp4", listener.Addr().String())
	require.NoError(t, err)

	// Let the stream finish before Shutdown: the proxy closes active streams
	// on Shutdown, which must not overlap with the handshake still going on.
	defer func() {
		conn.Close() //nolint: errcheck

		select {
		case <-finished:
		case <-time.After(5 * time.Second):
			t.Error("the stream has not finished")
		}
	}()

	client := conn.(*net.TCPConn) //nolint: forcetypeassert
	require.NoError(t, client.SetDeadline(time.Now().Add(5*time.Second)))

	before := clientTCPInfo(t, client)

	_, err = client.Write(snapshotClientHello(t))
	require.NoError(t, err)

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

	after := clientTCPInfo(t, client)

	return int(after.Segs_in - before.Segs_in), total
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

// With ServerHelloMSS the proxy sends the ServerHello in small segments (80
// bytes of payload here), without it - in a few large ones.
func TestProxyServerHelloMSS(t *testing.T) {
	plainSegs, plainSize := serverHelloSegments(t, 0)
	shapedSegs, shapedSize := serverHelloSegments(t, 92)

	assert.LessOrEqual(t, plainSegs, plainSize/1000+3)
	assert.GreaterOrEqual(t, shapedSegs, shapedSize/80)

	t.Logf("ServerHello without client-mss: %d bytes, %d segments; with client-mss 92: %d bytes, %d segments",
		plainSize, plainSegs, shapedSize, shapedSegs)
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
