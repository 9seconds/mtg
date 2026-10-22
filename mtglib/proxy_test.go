package mtglib_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
	"github.com/stretchr/testify/suite"
	"github.com/yl2chen/cidranger"
)

type ProxyTestSuite struct {
	suite.Suite

	opts     *mtglib.ProxyOpts
	p        *mtglib.Proxy
	listener net.Listener
}

func (suite *ProxyTestSuite) ProxyAddress() string {
	_, port, _ := net.SplitHostPort(suite.listener.Addr().String())

	return net.JoinHostPort("127.0.0.1", port)
}

func (suite *ProxyTestSuite) ProxySecret() string {
	return suite.opts.Secret.Hex()
}

func (suite *ProxyTestSuite) SetupSuite() {
	dialer, err := network.NewDefaultDialer(0, 0)
	suite.NoError(err)

	ntw, err := network.NewNetwork(dialer, "mtgtest", "1.1.1.1", 0)
	suite.NoError(err)

	allowlist, _ := ipblocklist.NewFireholFromFiles(
		logger.NewNoopLogger(),
		1,
		[]files.File{
			files.NewMem([]*net.IPNet{
				cidranger.AllIPv4,
				cidranger.AllIPv6,
			}),
		},
		nil,
	)

	go allowlist.Run(time.Second)

	suite.opts = &mtglib.ProxyOpts{
		Secret:          mtglib.GenerateSecret("httpbin.org"),
		Network:         ntw,
		AntiReplayCache: antireplay.NewNoop(),
		IPBlocklist:     ipblocklist.NewNoop(),
		IPAllowlist:     allowlist,
		EventStream:     events.NewNoopStream(),
		Logger:          logger.NewNoopLogger(),
		UseTestDCs:      true,
	}

	proxy, err := mtglib.NewProxy(*suite.opts)
	suite.NoError(err)

	suite.p = proxy

	listener, err := net.Listen("tcp", ":0")
	suite.NoError(err)

	suite.listener = listener

	go suite.p.Serve(suite.listener) //nolint: errcheck
}

func (suite *ProxyTestSuite) TearDownSuite() {
	if suite.listener != nil {
		suite.listener.Close() //nolint: errcheck
	}

	if suite.p != nil {
		suite.p.Shutdown()
	}
}

func (suite *ProxyTestSuite) TestCannotInitNoSecret() {
	opts := *suite.opts
	opts.Secret = mtglib.Secret{}

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoNetwork() {
	opts := *suite.opts
	opts.Network = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoAntiReplayCache() {
	opts := *suite.opts
	opts.AntiReplayCache = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoIPBlocklist() {
	opts := *suite.opts
	opts.IPBlocklist = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoIPAllowlist() {
	opts := *suite.opts
	opts.IPAllowlist = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoEventStream() {
	opts := *suite.opts
	opts.EventStream = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoLogger() {
	opts := *suite.opts
	opts.Logger = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitIncorrectPreferIP() {
	opts := *suite.opts
	opts.PreferIP = "xxx"

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestDomainFrontingAddress() {
	suite.Equal("httpbin.org:443", suite.p.DomainFrontingAddress())
}

func (suite *ProxyTestSuite) TestHTTPSRequest() {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	addr := fmt.Sprintf("https://%s/headers", suite.ProxyAddress())

	resp, err := client.Get(addr) //nolint: noctx
	suite.Require().NoError(err)

	defer resp.Body.Close() //nolint: errcheck

	suite.Equal(http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	suite.NoError(err)

	jsonStruct := struct {
		Headers struct {
			TraceID string `json:"X-Amzn-Trace-Id"` //nolint: tagliatelle
		} `json:"headers"`
	}{}

	suite.NoError(json.Unmarshal(data, &jsonStruct))
	suite.NotEmpty(jsonStruct.Headers.TraceID)
}

func TestProxy(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProxyTestSuite{})
}

// startLocalFronting starts a fronting server on 127.0.0.1 that closes every
// connection, so that tests never reach the real fronting domain. It returns
// the port.
func startLocalFronting(t *testing.T) uint {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { listener.Close() }) //nolint: errcheck

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			conn.Close() //nolint: errcheck
		}
	}()

	return uint(listener.Addr().(*net.TCPAddr).Port) //nolint: forcetypeassert, gosec
}

// startPendingHandshakeProxy starts a proxy with a pending-handshake limit of
// 1 per IP that fronts to a local server and returns its address.
func startPendingHandshakeProxy(t *testing.T, log mtglib.Logger, stream mtglib.EventStream) string {
	t.Helper()

	dialer, err := network.NewDefaultDialer(0, 0)
	if err != nil {
		t.Fatal(err)
	}

	ntw, err := network.NewNetwork(dialer, "mtgtest", "1.1.1.1", 0)
	if err != nil {
		t.Fatal(err)
	}

	allowlist, _ := ipblocklist.NewFireholFromFiles(
		logger.NewNoopLogger(),
		1,
		[]files.File{files.NewMem([]*net.IPNet{cidranger.AllIPv4, cidranger.AllIPv6})},
		nil,
	)

	go allowlist.Run(time.Second)

	// The allowlist loads asynchronously; until then every connection is
	// rejected by it, which would look like a rejection by the limit.
	deadline := time.Now().Add(5 * time.Second)

	for !allowlist.Contains(net.ParseIP("127.0.0.1")) {
		if time.Now().After(deadline) {
			t.Fatal("the allowlist was not loaded in time")
		}

		time.Sleep(10 * time.Millisecond)
	}

	proxy, err := mtglib.NewProxy(mtglib.ProxyOpts{
		Secret:                 mtglib.GenerateSecret("httpbin.org"),
		Network:                ntw,
		AntiReplayCache:        antireplay.NewNoop(),
		IPBlocklist:            ipblocklist.NewNoop(),
		IPAllowlist:            allowlist,
		EventStream:            stream,
		Logger:                 log,
		UseTestDCs:             true,
		DomainFrontingHost:     "127.0.0.1",
		DomainFrontingPort:     startLocalFronting(t),
		PendingHandshakesPerIP: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	go proxy.Serve(listener) //nolint: errcheck

	t.Cleanup(func() {
		listener.Close() //nolint: errcheck
		proxy.Shutdown()
	})

	return listener.Addr().String()
}

// closedQuickly reports whether the proxy closes conn without waiting for the
// handshake timeout.
func closedQuickly(conn net.Conn) bool {
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond)) //nolint: errcheck

	_, err := conn.Read(make([]byte, 1))

	var netErr net.Error

	return err != nil && (!errors.As(err, &netErr) || !netErr.Timeout())
}

func dialProxy(t *testing.T, addr string) net.Conn {
	t.Helper()

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	return conn
}

// openUntilRejected opens silent connections one by one until one is closed
// and returns the admitted ones. No fixed sleeps - the proxy may pick a
// connection up with a delay under load.
func openUntilRejected(t *testing.T, addr string) []net.Conn {
	t.Helper()

	var admitted []net.Conn

	for range 10 {
		conn := dialProxy(t, addr)
		if closedQuickly(conn) {
			conn.Close() //nolint: errcheck

			return admitted
		}

		admitted = append(admitted, conn)
	}

	t.Fatal("connections over the pending-handshake limit must be closed")

	return nil
}

// A second silent connection from the same IP is closed right away when the
// pending-handshake limit is 1; once the first one goes away, a new connection
// is admitted (it stays open waiting for its handshake).
func TestProxyPendingHandshakeLimit(t *testing.T) {
	t.Parallel()

	addr := startPendingHandshakeProxy(t, logger.NewNoopLogger(), events.NewNoopStream())

	// With the limit of 1 exactly one silent connection must stay pending.
	admitted := openUntilRejected(t, addr)
	if len(admitted) != 1 {
		t.Fatalf("exactly one pending handshake must be admitted with limit 1, got %d", len(admitted))
	}

	// When the pending connection goes away, its slot is released. The proxy
	// notices the EOF asynchronously, so retry with a short pause.
	admitted[0].Close() //nolint: errcheck

	for range 10 {
		conn := dialProxy(t, addr)
		ok := !closedQuickly(conn)

		conn.Close() //nolint: errcheck

		if ok {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("after the pending connection is gone, a new one must be admitted")
}

// infoRecorder records Info messages.
type infoRecorder struct {
	mu   sync.Mutex
	msgs []string
}

func (l *infoRecorder) Named(_ string) mtglib.Logger          { return l }
func (l *infoRecorder) BindInt(_ string, _ int) mtglib.Logger { return l }
func (l *infoRecorder) BindStr(_, _ string) mtglib.Logger     { return l }
func (l *infoRecorder) BindJSON(_, _ string) mtglib.Logger    { return l }
func (l *infoRecorder) Printf(_ string, _ ...any)             {}
func (l *infoRecorder) InfoError(_ string, _ error)           {}
func (l *infoRecorder) Warning(_ string)                      {}
func (l *infoRecorder) WarningError(_ string, _ error)        {}
func (l *infoRecorder) Debug(_ string)                        {}
func (l *infoRecorder) DebugError(_ string, _ error)          {}

func (l *infoRecorder) Info(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.msgs = append(l.msgs, msg)
}

func (l *infoRecorder) count(msg string) int {
	l.mu.Lock()
	defer l.mu.Unlock()

	n := 0

	for _, v := range l.msgs {
		if v == msg {
			n++
		}
	}

	return n
}

// rejectionCounter counts pending-handshake rejections.
type rejectionCounter struct {
	rejected atomic.Int32
}

func (r *rejectionCounter) Send(_ context.Context, evt mtglib.Event) {
	if typed, ok := evt.(mtglib.EventPendingHandshakeLimit); ok && typed.Action == mtglib.PendingHandshakeRejected {
		r.rejected.Add(1)
	}
}

// Under a flood every rejected connection must still be counted by the event
// (it is the metric), but the log line must be rate limited.
func TestProxyPendingHandshakeRejectionLogIsRateLimited(t *testing.T) {
	t.Parallel()

	log := &infoRecorder{}
	stream := &rejectionCounter{}
	addr := startPendingHandshakeProxy(t, log, stream)

	admitted := openUntilRejected(t, addr)
	defer func() {
		for _, conn := range admitted {
			conn.Close() //nolint: errcheck
		}
	}()

	const extra = 5

	for range extra {
		conn := dialProxy(t, addr)
		if !closedQuickly(conn) {
			t.Fatal("a connection over the limit must be closed")
		}

		conn.Close() //nolint: errcheck
	}

	if got := stream.rejected.Load(); got != extra+1 {
		t.Fatalf("every rejection must be reported as an event: want %d, got %d", extra+1, got)
	}

	if got := log.count("too many pending handshakes from this ip"); got != 1 {
		t.Fatalf("the rejection log line must be rate limited: want 1, got %d", got)
	}
}
