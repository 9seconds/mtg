package relay

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/9seconds/mtg/v2/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ConnTestSuite struct {
	suite.Suite

	ctxCancel   context.CancelFunc
	connMock    *testlib.NetConnMock
	tickChannel chan struct{}
	buf         []byte
	c           conn
}

func (suite *ConnTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())

	suite.tickChannel = make(chan struct{}, 1)
	suite.connMock = &testlib.NetConnMock{}
	suite.ctxCancel = cancel
	suite.buf = make([]byte, 5)

	suite.c = conn{
		ReadWriteCloser: suite.connMock,
		ctx:             ctx,
		tickChannel:     suite.tickChannel,
	}
}

func (suite *ConnTestSuite) TestReadOk() {
	suite.connMock.On("Read", mock.Anything).Once().Return(len(suite.buf), nil)

	n, err := suite.c.Read(suite.buf)
	suite.NoError(err)
	suite.Equal(len(suite.buf), n)

	select {
	case <-suite.tickChannel:
	default:
		suite.FailNow("cannot find a tick event")
	}
}

func (suite *ConnTestSuite) TestReadErr() {
	suite.connMock.On("Read", mock.Anything).Once().Return(0, io.EOF)

	_, err := suite.c.Read(suite.buf)
	suite.True(errors.Is(err, io.EOF))

	select {
	case <-suite.tickChannel:
	default:
		suite.FailNow("cannot find a tick event")
	}
}

func (suite *ConnTestSuite) TestReadContextDone() {
	suite.connMock.On("Read", mock.Anything).Once().Return(len(suite.buf), nil)
	suite.ctxCancel()

	suite.tickChannel <- struct{}{}

	suite.c.Read(suite.buf)
}

func (suite *ConnTestSuite) TestWriteOk() {
	suite.connMock.On("Write", mock.Anything).Once().Return(len(suite.buf), nil)

	n, err := suite.c.Write(suite.buf)
	suite.NoError(err)
	suite.Equal(len(suite.buf), n)

	select {
	case <-suite.tickChannel:
	default:
		suite.FailNow("cannot find a tick event")
	}
}

func (suite *ConnTestSuite) TestWriteErr() {
	suite.connMock.On("Write", mock.Anything).Once().Return(0, io.EOF)

	_, err := suite.c.Write(suite.buf)
	suite.True(errors.Is(err, io.EOF))

	select {
	case <-suite.tickChannel:
	default:
		suite.FailNow("cannot find a tick event")
	}
}

func (suite *ConnTestSuite) TestWriteContextDone() {
	suite.connMock.On("Write", mock.Anything).Once().Return(len(suite.buf), nil)
	suite.ctxCancel()

	suite.tickChannel <- struct{}{}

	suite.c.Write(suite.buf)
}

func (suite *ConnTestSuite) TearDownTest() {
	select {
	case <-suite.tickChannel:
	default:
	}

	close(suite.tickChannel)

	suite.connMock.AssertExpectations(suite.T())
}

func TestConn(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnTestSuite{})
}
