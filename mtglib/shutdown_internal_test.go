package mtglib

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls/fake"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type shutdownTestEvents struct{}

func (shutdownTestEvents) Send(context.Context, Event) {}

type shutdownAntiReplay struct{}

func (shutdownAntiReplay) SeenBefore([]byte) bool { return false }

type shutdownBlocklist struct{}

func (shutdownBlocklist) Contains(net.IP) bool { return false }
func (shutdownBlocklist) Run(time.Duration)    {}
func (shutdownBlocklist) Shutdown()            {}

// shutdownConn supplies ClientHello, then blocks reads on the pipe. It signals
// when the test should shut down the proxy, optionally at ServerHello instead.
type shutdownConn struct {
	essentials.Conn
	hello  *bytes.Reader
	ready  chan struct{}
	cancel context.CancelFunc
	once   sync.Once
}

func (c *shutdownConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}
}

func (c *shutdownConn) Read(p []byte) (int, error) {
	if c.hello.Len() != 0 {
		return c.hello.Read(p)
	}
	if c.cancel == nil {
		c.once.Do(func() { close(c.ready) })
	}

	return c.Conn.Read(p)
}

func (c *shutdownConn) Write(p []byte) (int, error) {
	if c.cancel != nil {
		c.cancel()
		c.once.Do(func() { close(c.ready) })
		// Allow shutdown to overlap installation of the TLS wrapper.
		runtime.Gosched()
	}

	return len(p), nil
}

func shutdownClientHello(t *testing.T) (Secret, []byte) {
	t.Helper()

	secret, err := ParseSecret("ee367a189aee18fa31c190054efd4a8e9573746f726167652e676f6f676c65617069732e636f6d")
	require.NoError(t, err)
	data, err := os.ReadFile("internal/tls/fake/testdata/client-hello-ok-19dfe38384b9884b.json")
	require.NoError(t, err)
	var snapshot struct{ Full string }
	require.NoError(t, json.Unmarshal(data, &snapshot))
	hello, err := base64.StdEncoding.DecodeString(snapshot.Full)
	require.NoError(t, err)
	random := hello[fake.RandomOffset : fake.RandomOffset+fake.RandomLen]
	clear(random)
	mac := hmac.New(sha256.New, secret.Key[:])
	mac.Write(hello)
	copy(random, mac.Sum(nil))
	timestamp := uint32(time.Now().Unix()) //nolint: gosec
	binary.LittleEndian.PutUint32(random[fake.RandomLen-4:], binary.LittleEndian.Uint32(random[fake.RandomLen-4:])^timestamp)

	return secret, hello
}

func TestServeConnShutdown(t *testing.T) {
	for _, stage := range []string{"before-faketls", "server-hello", "after-faketls"} {
		t.Run(stage, func(t *testing.T) {
			secret, hello := shutdownClientHello(t)
			if stage == "before-faketls" {
				hello = nil
			}
			ntw := &testlib.MtglibNetworkMock{}
			ntw.On("MakeHTTPClient", mock.Anything).Return(&http.Client{}).Once()
			ntw.On("NativeDialer").Return(&net.Dialer{Timeout: time.Second}).Maybe()
			proxy, err := NewProxy(ProxyOpts{
				Secret: secret, Network: ntw, Logger: NoopLogger{},
				EventStream: shutdownTestEvents{}, AntiReplayCache: shutdownAntiReplay{},
				IPBlocklist: shutdownBlocklist{}, IPAllowlist: shutdownBlocklist{},
				DomainFrontingHost: "127.0.0.1", DomainFrontingPort: 1,
				HandshakeTimeout: time.Hour, TolerateTimeSkewness: time.Minute,
			})
			require.NoError(t, err)
			server, client := net.Pipe()
			t.Cleanup(func() {
				server.Close() //nolint: errcheck
				client.Close() //nolint: errcheck
				proxy.Shutdown()
				ntw.AssertExpectations(t)
			})
			require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
			conn := &shutdownConn{
				Conn: essentials.WrapNetConn(server), hello: bytes.NewReader(hello),
				ready: make(chan struct{}),
			}
			if stage == "server-hello" {
				// Isolate the TLS wrapper race from doppelganger's NewConn shutdown.
				proxy.doppelGanger.Shutdown()
				conn.cancel = proxy.ctxCancel
			}
			go proxy.ServeConn(conn)
			select {
			case <-conn.ready:
			case <-time.After(5 * time.Second):
				t.Fatal("handshake did not reach the shutdown stage")
			}
			stopped := make(chan struct{})
			go func() {
				defer close(stopped)
				proxy.Shutdown()
			}()
			select {
			case <-stopped:
			case <-time.After(5 * time.Second):
				t.Fatal("shutdown did not interrupt the handshake")
			}
			_, err = client.Read(make([]byte, 1))
			require.ErrorIs(t, err, io.EOF)
		})
	}
}
