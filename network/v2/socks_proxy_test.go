package network_test

import (
	"net"
	"net/url"
	"sync"
	"testing"

	"github.com/9seconds/mtg/v2/network/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/things-go/go-socks5"
)

type SocksProxyTestSuite struct {
	EchoServerTestSuite

	wg          sync.WaitGroup
	baseNetwork network.Network

	noAuthURL *url.URL
	authURL   *url.URL

	noAuthListener net.Listener
	authListener   net.Listener

	noAuthServer *socks5.Server
	authServer   *socks5.Server
}

func (suite *SocksProxyTestSuite) SetupSuite() {
	suite.EchoServerTestSuite.SetupSuite()

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(suite.T(), err)
	suite.noAuthListener = listener

	listener, err = net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(suite.T(), err)
	suite.authListener = listener

	suite.noAuthServer = socks5.NewServer()
	suite.wg.Go(func() {
		suite.noAuthServer.Serve(suite.noAuthListener)
	})

	suite.authServer = socks5.NewServer(
		socks5.WithAuthMethods([]socks5.Authenticator{
			socks5.UserPassAuthenticator{
				Credentials: socks5.StaticCredentials{
					"user": "pass",
				},
			},
		}))
	suite.wg.Go(func() {
		suite.authServer.Serve(suite.authListener)
	})

	parsed, err := url.Parse("socks5://" + suite.noAuthListener.Addr().String())
	require.NoError(suite.T(), err)
	suite.noAuthURL = parsed

	parsed, err = url.Parse("socks5://user:pass@" + suite.authListener.Addr().String())
	require.NoError(suite.T(), err)
	suite.authURL = parsed

	suite.baseNetwork = network.New(nil, "mtg", 0, 0, 0)
}

func (suite *SocksProxyTestSuite) TestIncorrectSchema() {
	parsed, err := url.Parse("http://hello")
	suite.NoError(err)

	_, err = network.NewProxyNetwork(suite.baseNetwork, parsed)
	suite.Error(err)
}

func (suite *SocksProxyTestSuite) TestRead() {
	testData := map[string][]*url.URL{
		"noAuth": {suite.noAuthURL},
		"auth":   {suite.authURL},
		"both":   {suite.noAuthURL, suite.authURL},
	}

	for name, proxies := range testData {
		suite.T().Run(name, func(t *testing.T) {
			proxyNetworks := []network.Network{}

			for _, u := range proxies {
				value, err := network.NewProxyNetwork(suite.baseNetwork, u)
				assert.NoError(t, err)
				proxyNetworks = append(proxyNetworks, value)
			}

			netw, err := network.Join(proxyNetworks...)
			assert.NoError(t, err)

			conn, err := netw.Dial("tcp4", suite.EchoServerAddr())
			assert.NoError(t, err)

			data := []byte{1, 2, 3}
			n, err := conn.Write(data)
			assert.NoError(t, err)
			assert.Equal(t, len(data), n)

			toRead := []byte{1, 2, 3, 4, 5}
			n, err = conn.Read(toRead)
			assert.NoError(t, err)
			assert.Equal(t, len(data), n)
			assert.Equal(t, data, toRead[:n])
			assert.NotEqual(t, data, toRead)
		})
	}
}

func (suite *SocksProxyTestSuite) TearDownSuite() {
	suite.noAuthListener.Close()
	suite.authListener.Close()
	suite.wg.Wait()
	suite.EchoServerTestSuite.TearDownSuite()
}

func TestSocksProxy(t *testing.T) {
	t.Parallel()
	suite.Run(t, &SocksProxyTestSuite{})
}
