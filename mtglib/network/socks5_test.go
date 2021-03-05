package network_test

import (
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/network"
	socks5 "github.com/armon/go-socks5"
	"github.com/stretchr/testify/suite"
)

type Socks5TestSuite struct {
	HTTPServerTestSuite

	socksListener net.Listener
	socksProxy    *socks5.Server
}

func (suite *Socks5TestSuite) SetupSuite() {
	suite.HTTPServerTestSuite.SetupSuite()

	socksConf := socks5.Config{
		Credentials: socks5.StaticCredentials{
			"user": "password",
		},
	}

	suite.socksProxy, _ = socks5.New(&socksConf)
	suite.socksListener, _ = net.Listen("tcp", "127.0.0.1:0")

	go suite.socksProxy.Serve(suite.socksListener)
}

func (suite *Socks5TestSuite) TearDownSuite() {
	suite.socksListener.Close()

	suite.HTTPServerTestSuite.TearDownSuite()
}

func (suite *Socks5TestSuite) TestRequestFailed() {
	proxyURL := &url.URL{
		Scheme: "socks5",
		User:   url.UserPassword("user2", "password"),
		Host:   suite.socksListener.Addr().String(),
	}
	dialer, _ := network.NewSocks5Dialer(proxyURL, 0, 0)

	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}

	_, err := httpClient.Get(suite.httpServer.URL + "/get")

	suite.Error(err)
}

func (suite *Socks5TestSuite) TestRequestOk() {
	proxyURL := &url.URL{
		Scheme: "socks5",
		User:   url.UserPassword("user", "password"),
		Host:   suite.socksListener.Addr().String(),
	}
	dialer, _ := network.NewSocks5Dialer(proxyURL, 0, 0)

	httpClient := http.Client{
		Transport: &http.Transport{
			DialContext: dialer.DialContext,
		},
	}

	resp, err := httpClient.Get(suite.httpServer.URL + "/get")

	suite.NoError(err)

	resp.Body.Close()
}

func TestSocks5TestSuite(t *testing.T) {
	suite.Run(t, &Socks5TestSuite{})
}
