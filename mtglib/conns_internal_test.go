package mtglib

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ConnTrafficTestSuite struct {
	suite.Suite

	eventStreamMock *EventStreamMock
	connMock        *testlib.NetConnMock
	conn            io.ReadWriter
}

func (suite *ConnTrafficTestSuite) SetupTest() {
	suite.eventStreamMock = &EventStreamMock{}
	suite.connMock = &testlib.NetConnMock{}
	suite.conn = connTraffic{
		Conn:   suite.connMock,
		connID: "CONNID",
		ctx:    context.Background(),
		stream: suite.eventStreamMock,
	}
}

func (suite *ConnTrafficTestSuite) TearDownTest() {
	suite.eventStreamMock.AssertExpectations(suite.T())
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *ConnTrafficTestSuite) TestReadOk() {
	suite.eventStreamMock.
		On("Send", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			evt := args.Get(1).(EventTraffic)

			suite.Equal("CONNID", evt.StreamID())
			suite.WithinDuration(time.Now(), evt.Timestamp(), time.Second)
			suite.EqualValues(10, evt.Traffic)
			suite.True(evt.IsRead)
		})
	suite.connMock.On("Read", mock.Anything).Once().Return(10, nil)

	n, err := suite.conn.Read(make([]byte, 10))
	suite.NoError(err)
	suite.Equal(10, n)
}

func (suite *ConnTrafficTestSuite) TestReadErr() {
	suite.eventStreamMock.
		On("Send", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			evt := args.Get(1).(EventTraffic)

			suite.Equal("CONNID", evt.StreamID())
			suite.WithinDuration(time.Now(), evt.Timestamp(), time.Second)
			suite.EqualValues(10, evt.Traffic)
			suite.True(evt.IsRead)
		})
	suite.connMock.On("Read", mock.Anything).Once().Return(10, io.EOF)

	n, err := suite.conn.Read(make([]byte, 10))
	suite.True(errors.Is(err, io.EOF))
	suite.Equal(10, n)
}

func (suite *ConnTrafficTestSuite) TestReadNothingOk() {
	suite.connMock.On("Read", mock.Anything).Once().Return(0, nil)

	n, err := suite.conn.Read(make([]byte, 10))
	suite.NoError(err)
	suite.Equal(0, n)
}

func (suite *ConnTrafficTestSuite) TestReadNothingErr() {
	suite.connMock.On("Read", mock.Anything).Once().Return(0, io.EOF)

	n, err := suite.conn.Read(make([]byte, 10))
	suite.True(errors.Is(err, io.EOF))
	suite.Equal(0, n)
}

func (suite *ConnTrafficTestSuite) TestWriteOk() {
	suite.eventStreamMock.
		On("Send", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			evt := args.Get(1).(EventTraffic)

			suite.Equal("CONNID", evt.StreamID())
			suite.WithinDuration(time.Now(), evt.Timestamp(), time.Second)
			suite.EqualValues(10, evt.Traffic)
			suite.False(evt.IsRead)
		})
	suite.connMock.On("Write", mock.Anything).Once().Return(10, nil)

	n, err := suite.conn.Write(make([]byte, 10))
	suite.NoError(err)
	suite.Equal(10, n)
}

func (suite *ConnTrafficTestSuite) TestWriteErr() {
	suite.eventStreamMock.
		On("Send", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			evt := args.Get(1).(EventTraffic)

			suite.Equal("CONNID", evt.StreamID())
			suite.WithinDuration(time.Now(), evt.Timestamp(), time.Second)
			suite.EqualValues(10, evt.Traffic)
			suite.False(evt.IsRead)
		})
	suite.connMock.On("Write", mock.Anything).Once().Return(10, io.EOF)

	n, err := suite.conn.Write(make([]byte, 10))
	suite.True(errors.Is(err, io.EOF))
	suite.Equal(10, n)
}

func (suite *ConnTrafficTestSuite) TestWriteNothingOk() {
	suite.connMock.On("Write", mock.Anything).Once().Return(0, nil)

	n, err := suite.conn.Write(make([]byte, 10))
	suite.NoError(err)
	suite.Equal(0, n)
}

func (suite *ConnTrafficTestSuite) TestWriteNothingErr() {
	suite.connMock.On("Write", mock.Anything).Once().Return(0, io.EOF)

	n, err := suite.conn.Write(make([]byte, 10))
	suite.True(errors.Is(err, io.EOF))
	suite.Equal(0, n)
}

func TestConnTraffic(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnTrafficTestSuite{})
}
