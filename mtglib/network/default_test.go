package network_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/network"
	"github.com/mccutchen/go-httpbin/httpbin"
	"github.com/stretchr/testify/suite"
)

type DefaultDialerTestSuite struct {
	suite.Suite

	d          network.Dialer
	srvAddress string
	srv        *httptest.Server
}

func (suite *DefaultDialerTestSuite) SetupSuite() {
	suite.srv = httptest.NewServer(httpbin.NewHTTPBin().Handler())
	suite.srvAddress = strings.TrimPrefix(suite.srv.URL, "http://")
}

func (suite *DefaultDialerTestSuite) SetupTest() {
	d, err := network.NewDefaultDialer(0, 0)

	suite.NoError(err)

	suite.d = d
}

func (suite *DefaultDialerTestSuite) TearDownSuite() {
	suite.srv.Close()
}

func (suite *DefaultDialerTestSuite) TestNegativeTimeout() {
	_, err := network.NewDefaultDialer(-1, 0)

	suite.Error(err)
}

func (suite *DefaultDialerTestSuite) TestNegativeBufferSize() {
	_, err := network.NewDefaultDialer(0, -1)

	suite.Error(err)
}

func (suite *DefaultDialerTestSuite) TestUnsupportedProtocol() {
	_, err := suite.d.DialContext(context.Background(), "udp", suite.srvAddress)

	suite.Error(err)
}

func (suite *DefaultDialerTestSuite) TestCannotDial() {
	_, err := suite.d.DialContext(context.Background(),
		"tcp",
		suite.srvAddress+suite.srvAddress)

	suite.Error(err)
}

func (suite *DefaultDialerTestSuite) TestConnectOk() {
	conn, err := suite.d.DialContext(context.Background(),
		"tcp",
		suite.srvAddress)

	suite.NoError(err)
	suite.NotNil(conn)

	conn.Close()
}

func (suite *DefaultDialerTestSuite) TestRequest() {
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: suite.d.DialContext,
		},
	}

	resp, err := httpClient.Get(suite.srv.URL + "/get")

	suite.NoError(err)

	resp.Body.Close()
}

func TestDefaultDialer(t *testing.T) {
	suite.Run(t, &DefaultDialerTestSuite{})
}
