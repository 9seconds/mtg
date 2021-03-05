package network_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/network"
	"github.com/stretchr/testify/suite"
)

type DefaultDialerTestSuite struct {
	HTTPServerTestSuite

	d network.Dialer
}

func (suite *DefaultDialerTestSuite) SetupSuite() {
	suite.HTTPServerTestSuite.SetupSuite()

	d, err := network.NewDefaultDialer(0, 0)

	suite.NoError(err)

	suite.d = d
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
	_, err := suite.d.DialContext(context.Background(),
		"udp",
		suite.HTTPServerAddress())

	suite.Error(err)
}

func (suite *DefaultDialerTestSuite) TestCannotDial() {
	_, err := suite.d.DialContext(context.Background(),
		"tcp",
		suite.HTTPServerAddress()+suite.HTTPServerAddress())

	suite.Error(err)
}

func (suite *DefaultDialerTestSuite) TestConnectOk() {
	conn, err := suite.d.DialContext(context.Background(),
		"tcp",
		suite.HTTPServerAddress())

	suite.NoError(err)
	suite.NotNil(conn)

	conn.Close()
}

func (suite *DefaultDialerTestSuite) TestHTTPRequest() {
	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: suite.d.DialContext,
		},
	}

	resp, err := httpClient.Get(suite.httpServer.URL + "/get")

	suite.NoError(err)

	resp.Body.Close()
}

func TestDefaultDialer(t *testing.T) {
	suite.Run(t, &DefaultDialerTestSuite{})
}
