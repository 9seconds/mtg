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
	"sync/atomic"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/doppel"
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

type shutdownHandshakeConn struct {
	essentials.Conn
	input  *bytes.Reader
	cancel context.CancelFunc
	closed atomic.Int32
}

func (c *shutdownHandshakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (c *shutdownHandshakeConn) Read(p []byte) (int, error) {
	return c.input.Read(p)
}

func (c *shutdownHandshakeConn) Write(p []byte) (int, error) {
	// The server has sent ServerHello, but has not installed its TLS wrapper.
	c.cancel()
	runtime.Gosched()

	return len(p), nil
}

func (c *shutdownHandshakeConn) Close() error {
	err := c.Conn.Close()
	c.closed.Add(1)

	return err
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

func TestServeConnShutdownDuringServerHello(t *testing.T) {
	secret, hello := shutdownClientHello(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	server, client := net.Pipe()
	defer server.Close() //nolint: errcheck
	defer client.Close() //nolint: errcheck
	require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
	conn := &shutdownHandshakeConn{
		Conn: essentials.WrapNetConn(server), input: bytes.NewReader(hello), cancel: cancel,
	}
	p := &Proxy{
		ctx: ctx, ctxCancel: cancel, secret: secret,
		logger: NoopLogger{}, eventStream: shutdownTestEvents{},
		antiReplayCache:  shutdownAntiReplay{},
		handshakeTimeout: time.Second, tolerateTimeSkewness: time.Minute,
		doppelGanger: doppel.NewGanger(ctx, nil, NoopLogger{}, 0, 0, nil, false),
	}
	p.ServeConn(conn)
	require.Eventually(t, func() bool { return conn.closed.Load() >= 2 }, 5*time.Second, time.Millisecond)
	_, err := client.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF)
}

type shutdownBlockingConn struct {
	essentials.Conn
	input   *bytes.Reader
	reading chan struct{}
	once    sync.Once
}

func (c *shutdownBlockingConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 12345}
}

func (c *shutdownBlockingConn) Read(p []byte) (int, error) {
	if c.input.Len() != 0 {
		return c.input.Read(p)
	}

	c.once.Do(func() { close(c.reading) })

	return c.Conn.Read(p)
}

func (c *shutdownBlockingConn) Write(p []byte) (int, error) { return len(p), nil }

// Closing the original socket must interrupt reads both before FakeTLS and
// after the TLS and doppelganger wrappers have been installed.
func TestServeConnShutdownUnblocksHandshake(t *testing.T) {
	for _, afterFakeTLS := range []bool{false, true} {
		name := "before-faketls"
		if afterFakeTLS {
			name = "after-faketls"
		}

		t.Run(name, func(t *testing.T) {
			secret, hello := shutdownClientHello(t)
			if !afterFakeTLS {
				hello = nil
			}

			ntw := &testlib.MtglibNetworkMock{}
			ntw.On("MakeHTTPClient", mock.Anything).Return(&http.Client{}).Once()
			ntw.On("NativeDialer").Return(&net.Dialer{Timeout: time.Second}).Maybe()
			proxy, err := NewProxy(ProxyOpts{
				Secret: secret, Network: ntw, Logger: NoopLogger{},
				AntiReplayCache: shutdownAntiReplay{}, EventStream: shutdownTestEvents{},
				IPBlocklist: shutdownBlocklist{}, IPAllowlist: shutdownBlocklist{},
				DomainFrontingHost: "127.0.0.1", DomainFrontingPort: 1,
				HandshakeTimeout: time.Hour, TolerateTimeSkewness: time.Minute,
			})
			require.NoError(t, err)

			server, client := net.Pipe()
			require.NoError(t, client.SetReadDeadline(time.Now().Add(5*time.Second)))
			conn := &shutdownBlockingConn{
				Conn: essentials.WrapNetConn(server), input: bytes.NewReader(hello),
				reading: make(chan struct{}),
			}
			t.Cleanup(func() {
				server.Close() //nolint: errcheck
				client.Close() //nolint: errcheck
				proxy.Shutdown()
				ntw.AssertExpectations(t)
			})
			finished := make(chan struct{})
			go func() {
				defer close(finished)
				proxy.ServeConn(conn)
			}()

			select {
			case <-conn.reading:
			case <-time.After(5 * time.Second):
				t.Fatal("handshake did not reach the blocked read")
			}

			stopped := make(chan struct{})
			go func() {
				defer close(stopped)
				proxy.Shutdown()
			}()

			for _, done := range []<-chan struct{}{stopped, finished} {
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Fatal("shutdown did not interrupt the blocked handshake")
				}
			}

			_, err = client.Read(make([]byte, 1))
			require.ErrorIs(t, err, io.EOF)
		})
	}
}
