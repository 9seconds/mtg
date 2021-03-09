package network_test

import (
	"net/http/httptest"
	"strings"

	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/suite"
)

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
