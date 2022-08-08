package network_test

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/9seconds/mtg/v2/network"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type LoadBalancedSocks5TestSuite struct {
	suite.Suite
	HTTPServerTestSuite
	Socks5ServerTestSuite

	httpClient *http.Client
}

func (suite *LoadBalancedSocks5TestSuite) SetupSuite() {
	suite.HTTPServerTestSuite.SetupSuite()
	suite.Socks5ServerTestSuite.SetupSuite()
}

func (suite *LoadBalancedSocks5TestSuite) SetupTest() {
	baseDialer, _ := network.NewDefaultDialer(0, 0)
	lbDialer, err := network.NewLoadBalancedSocks5Dialer(baseDialer, []*url.URL{
		suite.MakeSocks5URL("user", "password"),
		suite.MakeSocks5URL("user2", "password"),
	})
	suite.NoError(err)

	suite.httpClient = suite.MakeHTTPClient(lbDialer)
}

func (suite *LoadBalancedSocks5TestSuite) TearDownSuite() {
	suite.Socks5ServerTestSuite.SetupSuite()
	suite.HTTPServerTestSuite.SetupSuite()
}

func (suite *LoadBalancedSocks5TestSuite) TestIncorrectURL() {
	_, err := network.NewLoadBalancedSocks5Dialer(&DialerMock{}, []*url.URL{
		{Scheme: "http"},
	})
	suite.Error(err)
}

func (suite *LoadBalancedSocks5TestSuite) TestCannotDial() {
	baseDialer := &DialerMock{}
	baseDialer.On("DialContext", mock.Anything, "tcp", "127.0.0.1:1080").
		Times(network.ProxyDialerOpenThreshold).
		Return(&net.TCPConn{}, io.EOF)
	baseDialer.On("DialContext", mock.Anything, "tcp", "127.0.0.2:1080").
		Times(network.ProxyDialerOpenThreshold).
		Return(&net.TCPConn{}, io.EOF)

	lbDialer, err := network.NewLoadBalancedSocks5Dialer(baseDialer, []*url.URL{
		{Scheme: "socks5", User: url.UserPassword("user", "password"), Host: "127.0.0.1:1080"},
		{Scheme: "socks5", User: url.UserPassword("user", "password"), Host: "127.0.0.2:1080"},
	})
	suite.NoError(err)

	for i := 0; i < network.ProxyDialerOpenThreshold*2; i++ {
		_, err = lbDialer.Dial("tcp", "127.1.1.1:80")
		suite.True(errors.Is(err, network.ErrCannotDialWithAllProxies))
	}

	baseDialer.AssertExpectations(suite.T())
}

func (suite *LoadBalancedSocks5TestSuite) TestDialOk() {
	resp, err := suite.httpClient.Get(suite.MakeURL("/get")) //nolint: noctx
	if err == nil {
		defer resp.Body.Close()
	}

	suite.NoError(err)
	suite.Equal(http.StatusOK, resp.StatusCode)
}

func TestLoadBalancedSocks5(t *testing.T) {
	t.Parallel()
	suite.Run(t, &LoadBalancedSocks5TestSuite{})
}
