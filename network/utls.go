package network

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"

	"github.com/9seconds/mtg/v2/essentials"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

type utlsRoundTripper struct {
	dialFunc  func(ctx context.Context, network, address string) (essentials.Conn, error)
	clientID  utls.ClientHelloID
	plainHTTP *http.Transport
}

func (u *utlsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "http" {
		return u.plainHTTP.RoundTrip(req) //nolint: wrapcheck
	}

	host := req.URL.Hostname()
	port := req.URL.Port()

	if port == "" {
		port = "443"
	}

	address := net.JoinHostPort(host, port)

	conn, err := u.dialTLS(req.Context(), address, host)
	if err != nil {
		return nil, fmt.Errorf("utls dial: %w", err)
	}

	var transport http.RoundTripper

	if conn.ConnectionState().NegotiatedProtocol == "h2" {
		transport = &http2.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
				return conn, nil
			},
		}
	} else {
		transport = &http.Transport{
			DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return conn, nil
			},
			DisableKeepAlives: true,
		}
	}

	resp, err := transport.RoundTrip(req)
	if err != nil {
		conn.Close() //nolint: errcheck

		return nil, err //nolint: wrapcheck
	}

	return resp, nil
}

func (u *utlsRoundTripper) dialTLS(
	ctx context.Context,
	address, serverName string,
) (*utls.UConn, error) {
	rawConn, err := u.dialFunc(ctx, "tcp", address)
	if err != nil {
		return nil, fmt.Errorf("dial: %w", err)
	}

	netConn, ok := rawConn.(net.Conn)
	if !ok {
		rawConn.Close() //nolint: errcheck

		return nil, fmt.Errorf("connection does not implement net.Conn")
	}

	uConn := utls.UClient(netConn, &utls.Config{
		ServerName: serverName,
	}, u.clientID)

	if err := uConn.HandshakeContext(ctx); err != nil {
		uConn.Close() //nolint: errcheck

		return nil, fmt.Errorf("utls handshake: %w", err)
	}

	return uConn, nil
}

func newUTLSTransport(
	dialFunc func(ctx context.Context, network, address string) (essentials.Conn, error),
) http.RoundTripper {
	return &utlsRoundTripper{
		dialFunc: dialFunc,
		clientID: utls.HelloChrome_Auto,
		plainHTTP: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialFunc(ctx, network, addr)
			},
		},
	}
}
