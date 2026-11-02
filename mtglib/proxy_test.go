package mtglib_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
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
	fronting *httptest.Server
}

func (suite *ProxyTestSuite) ProxyAddress() string {
	_, port, _ := net.SplitHostPort(suite.listener.Addr().String())

	return net.JoinHostPort("127.0.0.1", port)
}

func (suite *ProxyTestSuite) ProxySecret() string {
	return suite.opts.Secret.Hex()
}

func (suite *ProxyTestSuite) SetupSuite() {
	suite.fronting = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.Method+" "+r.URL.RequestURI()+" "+r.Header.Get("X-Mtg-Test"))
	}))
	host, port, err := net.SplitHostPort(suite.fronting.Listener.Addr().String())
	suite.Require().NoError(err)
	frontingPort, err := strconv.ParseUint(port, 10, 16)
	suite.Require().NoError(err)

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
	suite.T().Cleanup(allowlist.Shutdown)
	suite.Require().Eventually(func() bool {
		return allowlist.Contains(net.IPv4(127, 0, 0, 1))
	}, time.Second, time.Millisecond)

	suite.opts = &mtglib.ProxyOpts{
		Secret:             mtglib.GenerateSecret("fronting.example.test"),
		Network:            ntw,
		AntiReplayCache:    antireplay.NewNoop(),
		IPBlocklist:        ipblocklist.NewNoop(),
		IPAllowlist:        allowlist,
		EventStream:        events.NewNoopStream(),
		Logger:             logger.NewNoopLogger(),
		UseTestDCs:         true,
		DomainFrontingHost: host,
		DomainFrontingPort: uint(frontingPort),
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

	if suite.fronting != nil {
		suite.fronting.Close()
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
	suite.Equal(suite.fronting.Listener.Addr().String(), suite.p.DomainFrontingAddress())
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
	req, err := http.NewRequestWithContext(suite.T().Context(), http.MethodGet, addr, nil)
	suite.Require().NoError(err)
	req.Header.Set("X-Mtg-Test", "fronting-request")

	resp, err := client.Do(req)
	suite.Require().NoError(err)

	defer resp.Body.Close() //nolint: errcheck

	suite.Equal(http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	suite.NoError(err)

	suite.Equal("GET /headers fronting-request", string(data))
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

// startHoldingFronting starts a fronting server on 127.0.0.1 that keeps every
// connection open until the proxy closes its side or the test ends. Each
// accepted connection is reported to the returned channel. It returns the
// port.
func startHoldingFronting(t *testing.T) (uint, <-chan struct{}) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	accepted := make(chan struct{}, 16)
	done := make(chan struct{})

	t.Cleanup(func() {
		close(done)
		listener.Close() //nolint: errcheck
	})

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			select {
			case accepted <- struct{}{}:
			default:
			}

			go func() {
				<-done
				conn.Close() //nolint: errcheck
			}()

			go func() {
				io.Copy(io.Discard, conn) //nolint: errcheck
				conn.Close()              //nolint: errcheck
			}()
		}
	}()

	return uint(listener.Addr().(*net.TCPAddr).Port), accepted //nolint: forcetypeassert, gosec
}

// startPendingHandshakeProxy starts a proxy with a pending-handshake limit of
// 1 per IP that fronts to a local server and returns its address.
func startPendingHandshakeProxy(t *testing.T, log mtglib.Logger, stream mtglib.EventStream) string {
	t.Helper()

	return startPendingHandshakeProxyWith(
		t, log, stream, mtglib.GenerateSecret("httpbin.org"), startLocalFronting(t))
}

// startPendingHandshakeProxyWith is startPendingHandshakeProxy with the given
// secret and fronting port.
func startPendingHandshakeProxyWith(
	t *testing.T,
	log mtglib.Logger,
	stream mtglib.EventStream,
	secret mtglib.Secret,
	frontingPort uint,
) string {
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
		Secret:                 secret,
		Network:                ntw,
		AntiReplayCache:        antireplay.NewNoop(),
		IPBlocklist:            ipblocklist.NewNoop(),
		IPAllowlist:            allowlist,
		EventStream:            stream,
		Logger:                 log,
		UseTestDCs:             true,
		DomainFrontingHost:     "127.0.0.1",
		DomainFrontingPort:     frontingPort,
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

// Under a flood every rejected connection must still be counted by the event
// (it is the metric), but the log line must be rate limited.
func TestProxyPendingHandshakeRejectionLogIsRateLimited(t *testing.T) {
	t.Parallel()

	log := &infoRecorder{}
	stream := &eventRecorder{}
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

// makeFakeTLSClientHello returns a TLS record with a ClientHello that passes
// FakeTLS verification for the secret: a real crypto/tls ClientHello with the
// client random replaced by HMAC-SHA256(secret, hello with zero random) XOR
// the current timestamp.
func makeFakeTLSClientHello(t *testing.T, secret mtglib.Secret) []byte {
	t.Helper()

	clientSide, captureSide := net.Pipe()
	defer captureSide.Close() //nolint: errcheck

	go func() {
		tls.Client(clientSide, &tls.Config{ //nolint: gosec
			ServerName:         secret.Host,
			InsecureSkipVerify: true,
		}).Handshake() //nolint: errcheck

		clientSide.Close() //nolint: errcheck
	}()

	captureSide.SetDeadline(time.Now().Add(5 * time.Second)) //nolint: errcheck

	header := make([]byte, 5)
	if _, err := io.ReadFull(captureSide, header); err != nil {
		t.Fatal(err)
	}

	record := make([]byte, 5+int(binary.BigEndian.Uint16(header[3:])))
	copy(record, header)

	if _, err := io.ReadFull(captureSide, record[5:]); err != nil {
		t.Fatal(err)
	}

	// record_type(1) + version(2) + size(2) + handshake_type(1) +
	// uint24_length(3) + client_version(2)
	const randomOffset, randomLen = 11, 32

	random := record[randomOffset : randomOffset+randomLen]
	clear(random)

	mac := hmac.New(sha256.New, secret.Key[:])
	mac.Write(record)
	digest := mac.Sum(nil)

	timestamp := make([]byte, 4)
	binary.LittleEndian.PutUint32(timestamp, uint32(time.Now().Unix())) //nolint: gosec

	for i := range 4 {
		digest[randomLen-4+i] ^= timestamp[i]
	}

	copy(random, digest)

	return record
}

// fakeTLSHandshake connects to the proxy and sends a valid FakeTLS
// ClientHello. It reports whether the proxy answered with a ServerHello; if it
// did, the connection is returned open.
func fakeTLSHandshake(t *testing.T, addr string, secret mtglib.Secret) (net.Conn, bool) {
	t.Helper()

	conn := dialProxy(t, addr)

	if _, err := conn.Write(makeFakeTLSClientHello(t, secret)); err != nil {
		conn.Close() //nolint: errcheck

		return nil, false
	}

	conn.SetReadDeadline(time.Now().Add(5 * time.Second)) //nolint: errcheck

	header := make([]byte, 5)
	if _, err := io.ReadFull(conn, header); err != nil || header[0] != 0x16 {
		conn.Close() //nolint: errcheck

		return nil, false
	}

	conn.SetReadDeadline(time.Time{}) //nolint: errcheck

	return conn, true
}

// fakeTLSHandshakeAdmitted retries fakeTLSHandshake for a short while: the
// proxy frees the pending slot right after it sends the ServerHello, so the
// previous client may see its ServerHello a moment before the slot is free.
func fakeTLSHandshakeAdmitted(t *testing.T, addr string, secret mtglib.Secret) net.Conn {
	t.Helper()

	for range 20 {
		if conn, ok := fakeTLSHandshake(t, addr, secret); ok {
			return conn
		}

		time.Sleep(50 * time.Millisecond)
	}

	t.Fatal("a valid handshake from the same IP must be admitted")

	return nil
}

// stillOpen drains whatever the proxy has sent and reports whether conn is
// still open, i.e. reading times out instead of hitting EOF.
func stillOpen(conn net.Conn) bool {
	buf := make([]byte, 4096)

	for range 100 {
		conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond)) //nolint: errcheck

		if _, err := conn.Read(buf); err != nil {
			var netErr net.Error

			return errors.As(err, &netErr) && netErr.Timeout()
		}
	}

	return false
}

// eventRecorder counts the events the pending-handshake tests care about and
// keeps the contexts of started streams.
type eventRecorder struct {
	fronted  atomic.Int32
	rejected atomic.Int32

	mu      sync.Mutex
	streams []context.Context
}

func (r *eventRecorder) Send(ctx context.Context, evt mtglib.Event) {
	switch typed := evt.(type) {
	case mtglib.EventStart:
		r.mu.Lock()
		r.streams = append(r.streams, ctx)
		r.mu.Unlock()
	case mtglib.EventDomainFronting:
		r.fronted.Add(1)
	case mtglib.EventPendingHandshakeLimit:
		if typed.Action == mtglib.PendingHandshakeRejected {
			r.rejected.Add(1)
		}
	}
}

// waitStreamsDone waits until every started stream is over. A stream context
// is canceled when ServeConn returns, so this guarantees that the proxy is not
// shut down under a live session.
func (r *eventRecorder) waitStreamsDone(t *testing.T) {
	t.Helper()

	r.mu.Lock()
	streams := slices.Clone(r.streams)
	r.mu.Unlock()

	timeout := time.After(5 * time.Second)

	for _, ctx := range streams {
		select {
		case <-ctx.Done():
		case <-timeout:
			t.Error("streams are not finished in time")

			return
		}
	}
}

// An authenticated session does not hold a pending-handshake slot: with the
// limit of 1, a client that passed the FakeTLS handshake keeps its connection
// open, and another valid handshake from the same IP is still admitted.
func TestProxyPendingHandshakeLimitAuthenticatedSessionDoesNotHoldSlot(t *testing.T) {
	t.Parallel()

	secret := mtglib.GenerateSecret("httpbin.org")
	stream := &eventRecorder{}
	addr := startPendingHandshakeProxyWith(
		t, logger.NewNoopLogger(), stream, secret, startLocalFronting(t))

	// Runs before the proxy shutdown, after the connections are closed.
	t.Cleanup(func() { stream.waitStreamsDone(t) })

	first, ok := fakeTLSHandshake(t, addr, secret)
	if !ok {
		t.Fatal("the first valid handshake must be answered with a ServerHello")
	}
	defer first.Close() //nolint: errcheck

	second := fakeTLSHandshakeAdmitted(t, addr, secret)
	defer second.Close() //nolint: errcheck

	// The first session must still be alive: it was not closed to make room.
	if !stillOpen(first) {
		t.Fatal("the authenticated session must stay open")
	}

	if got := stream.fronted.Load(); got != 0 {
		t.Fatalf("valid handshakes must not be fronted, got %d fronting events", got)
	}
}

// A fronted connection does not hold a pending-handshake slot either: with the
// limit of 1, a client sent to the fronting domain keeps its connection open,
// and a valid handshake from the same IP is still admitted.
func TestProxyPendingHandshakeLimitFrontedConnectionDoesNotHoldSlot(t *testing.T) {
	t.Parallel()

	secret := mtglib.GenerateSecret("httpbin.org")
	frontingPort, fronted := startHoldingFronting(t)
	stream := &eventRecorder{}
	addr := startPendingHandshakeProxyWith(t, logger.NewNoopLogger(), stream, secret, frontingPort)

	// Runs before the proxy shutdown, after the connections are closed.
	t.Cleanup(func() { stream.waitStreamsDone(t) })

	// A ClientHello for another secret fails verification and is fronted.
	first := dialProxy(t, addr)
	defer first.Close() //nolint: errcheck

	if _, err := first.Write(makeFakeTLSClientHello(t, mtglib.GenerateSecret("httpbin.org"))); err != nil {
		t.Fatal(err)
	}

	select {
	case <-fronted:
	case <-time.After(5 * time.Second):
		t.Fatal("a foreign ClientHello must be sent to the fronting domain")
	}

	second := fakeTLSHandshakeAdmitted(t, addr, secret)
	defer second.Close() //nolint: errcheck

	// The fronted connection must still be alive: the backend holds it open.
	if !stillOpen(first) {
		t.Fatal("the fronted connection must stay open")
	}

	if got := stream.fronted.Load(); got != 1 {
		t.Fatalf("exactly one connection must be fronted, got %d", got)
	}
}
