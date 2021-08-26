package relay_test

import (
	"context"
	"io"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib/internal/relay"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type RelayTestSuite struct {
	suite.Suite

	loggerMock       relay.Logger
	ctx              context.Context
	ctxCancel        context.CancelFunc
	telegramConnMock *testlib.NetConnMock
	clientConnMock   *testlib.NetConnMock
}

func (suite *RelayTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	suite.ctx = ctx
	suite.ctxCancel = cancel
	suite.loggerMock = &loggerMock{}
	suite.telegramConnMock = &testlib.NetConnMock{}
	suite.clientConnMock = &testlib.NetConnMock{}
}

func (suite *RelayTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.telegramConnMock.AssertExpectations(suite.T())
	suite.clientConnMock.AssertExpectations(suite.T())
}

func (suite *RelayTestSuite) TestExit() {
	suite.telegramConnMock.On("SetReadDeadline", mock.Anything).Return(nil)
	suite.telegramConnMock.On("Close").Return(nil)
	suite.telegramConnMock.On("Read", mock.Anything).Return(10, io.EOF).Once()
	suite.telegramConnMock.On("Write", mock.Anything).Return(10, io.EOF).Maybe()

	suite.clientConnMock.On("Read", mock.Anything).Return(0, io.EOF).Once()
	suite.clientConnMock.On("Write", mock.Anything).Return(10, io.EOF).Maybe()
	suite.clientConnMock.On("Close").Return(nil)

	relay.Relay(suite.ctx, suite.loggerMock, 1024,
		suite.telegramConnMock, suite.clientConnMock)
}

func TestRelay(t *testing.T) {
	t.Parallel()
	suite.Run(t, &RelayTestSuite{})
}
