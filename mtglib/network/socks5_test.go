package network_test

import (
	"net/http"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/network"
	"github.com/stretchr/testify/suite"
)

type Socks5TestSuite struct {
	suite.Suite
	HTTPServerTestSuite
	Socks5ServerTestSuite

	d network.Dialer
}

func (suite *Socks5TestSuite) SetupSuite() {
	suite.HTTPServerTestSuite.SetupSuite()
	suite.Socks5ServerTestSuite.SetupSuite()

	suite.d, _ = network.NewDefaultDialer(0, 0)
}

func (suite *Socks5TestSuite) TearDownSuite() {
	suite.Socks5ServerTestSuite.TearDownSuite()
	suite.HTTPServerTestSuite.TearDownSuite()
}

func (suite *Socks5TestSuite) TestRequestFailed() {
	proxyURL := suite.MakeSocks5URL("user2", "password")
	dialer, _ := network.NewSocks5Dialer(suite.d, proxyURL)
	httpClient := suite.MakeHTTPClient(dialer)

	_, err := httpClient.Get(suite.MakeURL("/get"))

	suite.Error(err)
}

func (suite *Socks5TestSuite) TestRequestOk() {
	proxyURL := suite.MakeSocks5URL("user", "password")
	dialer, _ := network.NewSocks5Dialer(suite.d, proxyURL)
	httpClient := suite.MakeHTTPClient(dialer)

	resp, err := httpClient.Get(suite.MakeURL("/get"))

	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func TestSocks5TestSuite(t *testing.T) {
	suite.Run(t, &Socks5TestSuite{})
}
