package network_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/network"
	socks5 "github.com/armon/go-socks5"
	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/mock"
)

type DialerMock struct {
	mock.Mock
}

func (d *DialerMock) Dial(network, address string) (essentials.Conn, error) {
	args := d.Called(network, address)

	return args.Get(0).(essentials.Conn), args.Error(1) //nolint: wrapcheck, forcetypeassert
}

func (d *DialerMock) DialContext(ctx context.Context, network, address string) (essentials.Conn, error) {
	args := d.Called(ctx, network, address)

	return args.Get(0).(essentials.Conn), args.Error(1) //nolint: wrapcheck, forcetypeassert
}

type HTTPServerTestSuite struct {
	httpServer *httptest.Server
}

func (suite *HTTPServerTestSuite) SetupSuite() {
	suite.httpServer = httptest.NewServer(httpbin.NewHTTPBin().Handler())
}

func (suite *HTTPServerTestSuite) TearDownSuite() {
	suite.httpServer.Close()
}

func (suite *HTTPServerTestSuite) HTTPServerAddress() string {
	return strings.TrimPrefix(suite.httpServer.URL, "http://")
}

func (suite *HTTPServerTestSuite) MakeURL(path string) string {
	return suite.httpServer.URL + path
}

func (suite *HTTPServerTestSuite) MakeHTTPClient(dialer network.Dialer) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
				return dialer.DialContext(ctx, network, address) //nolint: wrapcheck
			},
		},
	}
}

type Socks5ServerTestSuite struct {
	socks5Listener net.Listener
	socks5Server   *socks5.Server
}

func (suite *Socks5ServerTestSuite) SetupSuite() {
	suite.socks5Listener, _ = net.Listen("tcp", "127.0.0.1:0")
	suite.socks5Server, _ = socks5.New(&socks5.Config{
		Credentials: socks5.StaticCredentials{
			"user": "password",
		},
	})

	go suite.socks5Server.Serve(suite.socks5Listener) //nolint: errcheck
}

func (suite *Socks5ServerTestSuite) TearDownSuite() {
	suite.socks5Listener.Close()
}

func (suite *Socks5ServerTestSuite) MakeSocks5URL(user, password string) *url.URL {
	return &url.URL{
		Scheme: "socks5",
		User:   url.UserPassword(user, password),
		Host:   suite.socks5Listener.Addr().String(),
	}
}
