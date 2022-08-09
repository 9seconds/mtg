package mtglib

import (
	"context"
	"net"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/stretchr/testify/suite"
)

type StreamContextTestSuite struct {
	suite.Suite

	connMock  *testlib.EssentialsConnMock
	logger    NoopLogger
	ctx       *streamContext
	ctxCancel context.CancelFunc
}

func (suite *StreamContextTestSuite) SetupSuite() {
	suite.logger = NoopLogger{}
}

func (suite *StreamContextTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "key", "value") //nolint: golint, staticcheck

	suite.ctxCancel = cancel
	suite.connMock = &testlib.EssentialsConnMock{}

	addr := &net.TCPAddr{
		IP:   net.ParseIP("10.0.0.10"),
		Port: 6676,
	}
	suite.connMock.On("RemoteAddr").Return(addr)

	suite.ctx = newStreamContext(ctx, suite.logger, suite.connMock)
}

func (suite *StreamContextTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *StreamContextTestSuite) TestContextInterface() {
	_, ok := suite.ctx.Deadline()
	suite.False(ok)

	select {
	case <-suite.ctx.Done():
		suite.FailNow("unexpectedly done")
	default:
	}

	suite.NoError(suite.ctx.Err())
	suite.Equal("value", suite.ctx.Value("key"))

	suite.ctxCancel()

	select {
	case <-suite.ctx.Done():
		suite.Error(suite.ctx.Err())
	default:
		suite.FailNow("unexpectedly not done")
	}
}

func (suite *StreamContextTestSuite) TestClientIP() {
	suite.Equal("10.0.0.10", suite.ctx.ClientIP().String())
}

func (suite *StreamContextTestSuite) TestClose() {
	suite.connMock.On("Close").Once().Return(nil)

	tgConnMock := &testlib.EssentialsConnMock{}
	tgConnMock.On("Close").Once().Return(nil)

	suite.ctx.telegramConn = tgConnMock
	suite.ctx.Close()

	select {
	case <-suite.ctx.Done():
		suite.Error(suite.ctx.Err())
	default:
		suite.FailNow("unexpectedly not done")
	}

	tgConnMock.AssertExpectations(suite.T())
}

func TestStreamContext(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StreamContextTestSuite{})
}
