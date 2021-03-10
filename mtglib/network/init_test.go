package network_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/network"
	socks5 "github.com/armon/go-socks5"
	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/mock"
)

type ConnMock struct {
	mock.Mock
}

func (c *ConnMock) Read(b []byte) (int, error) {
	args := c.Called(b)

	return args.Int(0), args.Error(1)
}

func (c *ConnMock) Write(b []byte) (int, error) {
	args := c.Called(b)

	return args.Int(0), args.Error(1)
}

func (c *ConnMock) Close() error {
	return c.Called().Error(0)
}

func (c *ConnMock) LocalAddr() net.Addr {
	return c.Called().Get(0).(net.Addr)
}

func (c *ConnMock) RemoteAddr() net.Addr {
	return c.Called().Get(0).(net.Addr)
}

func (c *ConnMock) SetDeadline(t time.Time) error {
	return c.Called(t).Error(0)
}

func (c *ConnMock) SetReadDeadline(t time.Time) error {
	return c.Called(t).Error(0)
}

func (c *ConnMock) SetWriteDeadline(t time.Time) error {
	return c.Called(t).Error(0)
}

type DialerMock struct {
	mock.Mock
}

func (d *DialerMock) Dial(network, address string) (net.Conn, error) {
	args := d.Called(network, address)

	return args.Get(0).(net.Conn), args.Error(1)
}

func (d *DialerMock) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	args := d.Called(ctx, network, address)

	return args.Get(0).(net.Conn), args.Error(1)
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
			DialContext: dialer.DialContext,
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

	go suite.socks5Server.Serve(suite.socks5Listener)
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
