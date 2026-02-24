package mtglib

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/pires/go-proxyproto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ConnRewindBaseConn struct {
	testlib.EssentialsConnMock

	readBuffer bytes.Buffer
}

func (c *ConnRewindBaseConn) Read(p []byte) (int, error) {
	c.Called(p)

	return c.readBuffer.Read(p) //nolint: wrapcheck
}

type ConnTrafficTestSuite struct {
	suite.Suite

	eventStreamMock *EventStreamMock
	connMock        *testlib.EssentialsConnMock
	conn            io.ReadWriter
}

func (suite *ConnTrafficTestSuite) SetupTest() {
	suite.eventStreamMock = &EventStreamMock{}
	suite.connMock = &testlib.EssentialsConnMock{}
	suite.conn = connTraffic{
		Conn:     suite.connMock,
		streamID: "CONNID",
		ctx:      context.Background(),
		stream:   suite.eventStreamMock,
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
			evt, ok := args.Get(1).(EventTraffic)

			suite.True(ok)
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

func (suite *ConnTrafficTestSuite) TestReadErr() { //nolint: dupl
	suite.eventStreamMock.
		On("Send", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			evt, ok := args.Get(1).(EventTraffic)

			suite.True(ok)
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
			evt, ok := args.Get(1).(EventTraffic)

			suite.True(ok)
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

func (suite *ConnTrafficTestSuite) TestWriteErr() { //nolint: dupl
	suite.eventStreamMock.
		On("Send", mock.Anything, mock.Anything).
		Once().
		Run(func(args mock.Arguments) {
			evt, ok := args.Get(1).(EventTraffic)

			suite.True(ok)
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

type ConnRewindTestSuite struct {
	suite.Suite

	connMock *ConnRewindBaseConn
	conn     *connRewind
}

func (suite *ConnRewindTestSuite) SetupTest() {
	suite.connMock = &ConnRewindBaseConn{}
	suite.conn = newConnRewind(suite.connMock)
}

func (suite *ConnRewindTestSuite) TearDownTest() {
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *ConnRewindTestSuite) TestRead() {
	suite.connMock.On("Read", mock.Anything)
	suite.connMock.readBuffer.Write([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	buf := make([]byte, 2)

	n, err := suite.conn.Read(buf)
	suite.NoError(err)
	suite.Equal(2, n)
	suite.Equal([]byte{1, 2}, buf)

	n, err = suite.conn.Read(buf)
	suite.NoError(err)
	suite.Equal(2, n)
	suite.Equal([]byte{3, 4}, buf)

	suite.conn.Rewind()

	data, err := io.ReadAll(suite.conn)
	suite.NoError(err)
	suite.Equal([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, data)
}

type ConnProxyProtocolTestSuite struct {
	suite.Suite

	sourceConnMock *testlib.EssentialsConnMock
	targetConnMock *testlib.EssentialsConnMock
	conn           *connProxyProtocol
}

func (suite *ConnProxyProtocolTestSuite) SetupTest() {
	suite.sourceConnMock = &testlib.EssentialsConnMock{}
	suite.targetConnMock = &testlib.EssentialsConnMock{}

	localAddr := &net.TCPAddr{
		IP: net.ParseIP("127.0.0.1").To4(),
	}
	remoteAddr := &net.TCPAddr{
		IP: net.ParseIP("127.0.0.2").To4(),
	}

	suite.sourceConnMock.
		On("RemoteAddr").
		Return(localAddr)
	suite.targetConnMock.
		On("RemoteAddr").
		Maybe().
		Return(remoteAddr)

	suite.conn = newConnProxyProtocol(suite.sourceConnMock, suite.targetConnMock)
}

func (suite *ConnProxyProtocolTestSuite) TestRead() {
	value := []byte{1, 2, 3, 4, 5}
	toRead := make([]byte, len(value))

	suite.targetConnMock.
		On("Read", mock.AnythingOfType("[]uint8")).
		Once().
		Return(len(toRead), nil).
		Run(func(args mock.Arguments) {
			arr := args.Get(0).([]byte)
			copy(arr, value)
		})

	n, err := suite.conn.Read(toRead)
	suite.Equal(len(value), n)
	suite.NoError(err)
	suite.Equal(value, toRead)
}

func (suite *ConnProxyProtocolTestSuite) TestWrite() {
	value := []byte{1, 2, 3, 4, 5}
	buf := &bytes.Buffer{}
	bufReader := bufio.NewReader(buf)

	suite.targetConnMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(28, nil).
		Run(func(args mock.Arguments) {
			arr := args.Get(0).([]byte)
			buf.Write(arr)
		})

	_, err := suite.conn.Write(value)
	suite.NoError(err)

	header, err := proxyproto.Read(bufReader)
	suite.NoError(err)

	sourceAddr, destAddr, ok := header.TCPAddrs()
	suite.True(ok)
	suite.Equal(suite.sourceConnMock.RemoteAddr(), sourceAddr)
	suite.Equal(suite.targetConnMock.RemoteAddr(), destAddr)

	read, _ := io.ReadAll(bufReader)
	suite.Equal(value, read)

	_, err = suite.conn.Write(value)
	suite.NoError(err)

	read, _ = io.ReadAll(bufReader)
	suite.Equal(value, read)
}

func (suite *ConnProxyProtocolTestSuite) TearDownTest() {
	suite.sourceConnMock.AssertExpectations(suite.T())
	suite.targetConnMock.AssertExpectations(suite.T())
}

func TestConnTraffic(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnTrafficTestSuite{})
}

func TestConnRewind(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnRewindTestSuite{})
}

func TestConnProxyProtocol(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnProxyProtocolTestSuite{})
}
