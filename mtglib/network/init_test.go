package network_test

import (
	"context"
	"net"
	"net/http/httptest"
	"strings"

	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

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
	suite.Suite

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
