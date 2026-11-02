package mtglib

import (
	"context"
	"math"
	"net"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/dc"
	"github.com/9seconds/mtg/v2/mtglib/obfuscation"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

const telegramCallTestTimeout = 5 * time.Second

type telegramHandshakeResult struct {
	dc  int
	err error
}

type ProxyTelegramCallTestSuite struct {
	suite.Suite

	networkMock     *testlib.MtglibNetworkMock
	eventStreamMock *EventStreamMock
	proxy           *Proxy
	ctx             *streamContext
	ctxCancel       context.CancelFunc

	telegramSide  net.Conn
	dialedAddress string
	telegramDC    chan telegramHandshakeResult
	connectedDC   chan int
}

func (suite *ProxyTelegramCallTestSuite) SetupTest() {
	tg, err := dc.New("only-ipv4")
	suite.Require().NoError(err)

	suite.networkMock = &testlib.MtglibNetworkMock{}
	suite.eventStreamMock = &EventStreamMock{}
	suite.dialedAddress = ""
	suite.telegramDC = make(chan telegramHandshakeResult, 1)
	suite.connectedDC = make(chan int, 1)

	// Pretend to be Telegram: accept the dial over an in-memory pipe and
	// decode the obfuscated handshake mtg sends there.
	proxySide, telegramSide := net.Pipe()
	deadline := time.Now().Add(telegramCallTestTimeout)
	suite.Require().NoError(proxySide.SetDeadline(deadline))
	suite.Require().NoError(telegramSide.SetDeadline(deadline))
	suite.T().Cleanup(func() {
		proxySide.Close() //nolint: errcheck
	})
	suite.telegramSide = telegramSide

	suite.networkMock.
		On("Dial", "tcp4", mock.AnythingOfType("string")).
		Maybe().
		Return(essentials.WrapNetConn(proxySide), nil).
		Run(func(args mock.Arguments) {
			suite.dialedAddress = args.String(1)

			go func() {
				dcid, _, err := obfuscation.Obfuscator{}.ReadHandshake(
					essentials.WrapNetConn(telegramSide))
				suite.telegramDC <- telegramHandshakeResult{dc: dcid, err: err}
			}()
		})

	suite.eventStreamMock.
		On("Send", mock.Anything, mock.AnythingOfType("mtglib.EventConnectedToDC")).
		Maybe().
		Run(func(args mock.Arguments) {
			suite.connectedDC <- args.Get(1).(EventConnectedToDC).DC //nolint: forcetypeassert
		})

	suite.proxy = &Proxy{
		telegram:    tg,
		network:     suite.networkMock,
		eventStream: suite.eventStreamMock,
		logger:      NoopLogger{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	suite.ctxCancel = cancel
	suite.ctx = &streamContext{
		ctx:       ctx,
		ctxCancel: cancel,
		streamID:  "stream",
		logger:    NoopLogger{},
	}
}

func (suite *ProxyTelegramCallTestSuite) TearDownTest() {
	if suite.ctx.telegramConn != nil {
		suite.ctx.telegramConn.Close() //nolint: errcheck
	}

	suite.telegramSide.Close() //nolint: errcheck
	suite.ctxCancel()
	suite.networkMock.AssertExpectations(suite.T())
	suite.eventStreamMock.AssertExpectations(suite.T())
}

func (suite *ProxyTelegramCallTestSuite) assertCall(requestedDC, expectedDC int) {
	suite.ctx.dc = requestedDC

	suite.Require().NoError(suite.proxy.doTelegramCall(suite.ctx))

	// address lookup goes by the absolute DC number
	suite.Contains(essentials.TelegramCoreAddresses[2], suite.dialedAddress)
	// the outgoing handshake and the connected-DC event keep the sign
	select {
	case result := <-suite.telegramDC:
		suite.Require().NoError(result.err, "decode outgoing Telegram handshake")
		suite.Equal(expectedDC, result.dc)
	case <-time.After(telegramCallTestTimeout):
		suite.FailNow("timed out waiting for outgoing Telegram handshake")
	}

	select {
	case connectedDC := <-suite.connectedDC:
		suite.Equal(expectedDC, connectedDC)
	case <-time.After(telegramCallTestTimeout):
		suite.FailNow("timed out waiting for connected-DC event")
	}
	suite.Equal(expectedDC, suite.ctx.dc)
}

func (suite *ProxyTelegramCallTestSuite) TestPositiveDC() {
	suite.assertCall(2, 2)
}

func (suite *ProxyTelegramCallTestSuite) TestNegativeDC() {
	suite.assertCall(-2, -2)
}

func (suite *ProxyTelegramCallTestSuite) TestPositiveUnknownDCFallback() {
	suite.proxy.allowFallbackOnUnknownDC = true

	suite.assertCall(42, dc.DefaultDC)
}

func (suite *ProxyTelegramCallTestSuite) TestNegativeUnknownDCFallback() {
	suite.proxy.allowFallbackOnUnknownDC = true

	suite.assertCall(-42, -dc.DefaultDC)
}

func (suite *ProxyTelegramCallTestSuite) TestInt16MinimumDCFallback() {
	suite.proxy.allowFallbackOnUnknownDC = true

	suite.assertCall(math.MinInt16, -dc.DefaultDC)
}

func (suite *ProxyTelegramCallTestSuite) TestUnknownDCWithoutFallback() {
	for _, dcid := range []int{42, -42, math.MinInt16} {
		suite.ctx.dc = dcid

		suite.Error(suite.proxy.doTelegramCall(suite.ctx))
		suite.Equal(dcid, suite.ctx.dc)
	}

	suite.networkMock.AssertNotCalled(suite.T(), "Dial", mock.Anything, mock.Anything)
	suite.eventStreamMock.AssertNotCalled(suite.T(), "Send", mock.Anything, mock.Anything)
}

func TestProxyTelegramCall(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ProxyTelegramCallTestSuite{})
}
