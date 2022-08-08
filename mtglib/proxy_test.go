package mtglib_test

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/9seconds/mtg/v2/events"
	"github.com/9seconds/mtg/v2/ipblocklist"
	"github.com/9seconds/mtg/v2/ipblocklist/files"
	"github.com/9seconds/mtg/v2/logger"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/9seconds/mtg/v2/network"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"github.com/stretchr/testify/suite"
	"github.com/yl2chen/cidranger"
)

type ProxyTestSuite struct {
	suite.Suite

	opts     *mtglib.ProxyOpts
	p        *mtglib.Proxy
	listener net.Listener
}

func (suite *ProxyTestSuite) ProxyAddress() string {
	_, port, _ := net.SplitHostPort(suite.listener.Addr().String())

	return net.JoinHostPort("127.0.0.1", port)
}

func (suite *ProxyTestSuite) ProxySecret() string {
	return suite.opts.Secret.Hex()
}

func (suite *ProxyTestSuite) SetupSuite() {
	dialer, err := network.NewDefaultDialer(0, 0)
	suite.NoError(err)

	ntw, err := network.NewNetwork(dialer, "mtgtest", "1.1.1.1", 0)
	suite.NoError(err)

	allowlist, _ := ipblocklist.NewFireholFromFiles(
		logger.NewNoopLogger(),
		1,
		[]files.File{
			files.NewMem([]*net.IPNet{
				cidranger.AllIPv4,
				cidranger.AllIPv6,
			}),
		},
		nil,
	)

	go allowlist.Run(time.Second)

	suite.opts = &mtglib.ProxyOpts{
		Secret:          mtglib.GenerateSecret("httpbin.org"),
		Network:         ntw,
		AntiReplayCache: antireplay.NewNoop(),
		IPBlocklist:     ipblocklist.NewNoop(),
		IPAllowlist:     allowlist,
		EventStream:     events.NewNoopStream(),
		Logger:          logger.NewNoopLogger(),
		UseTestDCs:      true,
	}

	proxy, err := mtglib.NewProxy(*suite.opts)
	suite.NoError(err)

	suite.p = proxy

	listener, err := net.Listen("tcp", ":0")
	suite.NoError(err)

	suite.listener = listener

	go suite.p.Serve(suite.listener) //nolint: errcheck
}

func (suite *ProxyTestSuite) TearDownSuite() {
	if suite.listener != nil {
		suite.listener.Close()
	}

	if suite.p != nil {
		suite.p.Shutdown()
	}
}

func (suite *ProxyTestSuite) TestCannotInitNoSecret() {
	opts := *suite.opts
	opts.Secret = mtglib.Secret{}

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoNetwork() {
	opts := *suite.opts
	opts.Network = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoAntiReplayCache() {
	opts := *suite.opts
	opts.AntiReplayCache = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoIPBlocklist() {
	opts := *suite.opts
	opts.IPBlocklist = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoIPAllowlist() {
	opts := *suite.opts
	opts.IPAllowlist = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoEventStream() {
	opts := *suite.opts
	opts.EventStream = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitNoLogger() {
	opts := *suite.opts
	opts.Logger = nil

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestCannotInitIncorrectPreferIP() {
	opts := *suite.opts
	opts.PreferIP = "xxx"

	_, err := mtglib.NewProxy(opts)
	suite.Error(err)
}

func (suite *ProxyTestSuite) TestDomainFrontingAddress() {
	suite.Equal("httpbin.org:443", suite.p.DomainFrontingAddress())
}

func (suite *ProxyTestSuite) TestHTTPSRequest() {
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
		Timeout: 5 * time.Second,
	}

	addr := fmt.Sprintf("https://%s/headers", suite.ProxyAddress())

	resp, err := client.Get(addr) //nolint: noctx
	suite.NoError(err)

	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	data, err := io.ReadAll(resp.Body)
	suite.NoError(err)

	jsonStruct := struct {
		Headers struct {
			TraceID string `json:"X-Amzn-Trace-Id"` //nolint: tagliatelle
		} `json:"headers"`
	}{}

	suite.NoError(json.Unmarshal(data, &jsonStruct))
	suite.NotEmpty(jsonStruct.Headers.TraceID)
}

func (suite *ProxyTestSuite) TestMakeRealRequest() {
	secret, _ := hex.DecodeString(suite.opts.Secret.Hex())
	resolver, err := dcs.MTProxyResolver(
		suite.ProxyAddress(),
		secret,
		dcs.MTProxyOptions{},
	)
	suite.NoError(err)

	tgClient := telegram.NewClient(telegram.TestAppID,
		telegram.TestAppHash,
		telegram.Options{
			Resolver: resolver,
		})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	suite.NoError(tgClient.Run(ctx, func(ctx context.Context) error {
		_, err := tg.NewClient(tgClient).HelpGetConfig(ctx)
		suite.NoError(err)

		return err //nolint: wrapcheck
	}))
}

func TestProxy(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProxyTestSuite{})
}
