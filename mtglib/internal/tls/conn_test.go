package tls

import (
	"io"
	"testing"

	"github.com/dolonet/mtg-multi/internal/testlib"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type ConnTestSuite struct {
	suite.Suite

	connMock *testlib.EssentialsConnMock
}

func (suite *ConnTestSuite) SetupTest() {
	suite.connMock = &testlib.EssentialsConnMock{}
}

func (suite *ConnTestSuite) TearDownTest() {
	suite.connMock.AssertExpectations(suite.T())
}

func (suite *ConnTestSuite) feedRead(raw []byte) {
	suite.connMock.
		On("Read", mock.AnythingOfType("[]uint8")).
		Run(func(args mock.Arguments) {
			copy(args.Get(0).([]byte), raw)
		}).
		Return(len(raw), nil).
		Once()
	suite.connMock.
		On("Read", mock.AnythingOfType("[]uint8")).
		Return(0, io.EOF).
		Maybe()
}

func (suite *ConnTestSuite) TestReadTLSEnabled() {
	payload := []byte("hello world")
	suite.feedRead(MakeTLSRecord(0x17, payload))

	conn := New(suite.connMock, true, false)

	buf := make([]byte, 128)
	n, err := conn.Read(buf)

	suite.NoError(err)
	suite.Equal(payload, buf[:n])
}

func (suite *ConnTestSuite) TestReadTLSSkipsNonApplicationData() {
	raw := append(
		MakeTLSRecord(0x14, []byte{1}),
		MakeTLSRecord(0x17, []byte("real data"))...,
	)
	suite.feedRead(raw)

	conn := New(suite.connMock, true, false)

	buf := make([]byte, 128)
	n, err := conn.Read(buf)

	suite.NoError(err)
	suite.Equal([]byte("real data"), buf[:n])
}

func (suite *ConnTestSuite) TestReadTLSMultipleRecords() {
	raw := append(
		MakeTLSRecord(0x17, []byte("first")),
		MakeTLSRecord(0x17, []byte("second"))...,
	)
	suite.feedRead(raw)

	conn := New(suite.connMock, true, false)
	buf := make([]byte, 128)

	n, err := conn.Read(buf)
	suite.NoError(err)
	suite.Equal([]byte("first"), buf[:n])

	n, err = conn.Read(buf)
	suite.NoError(err)
	suite.Equal([]byte("second"), buf[:n])
}

func (suite *ConnTestSuite) TestReadTLSSmallBuffer() {
	payload := []byte("hello world, this is a longer payload")
	suite.feedRead(MakeTLSRecord(0x17, payload))

	conn := New(suite.connMock, true, false)

	small := make([]byte, 5)
	n, err := conn.Read(small)
	suite.NoError(err)
	suite.Equal(payload[:5], small[:n])

	rest := make([]byte, 128)
	n, err = conn.Read(rest)
	suite.NoError(err)
	suite.Equal(payload[5:], rest[:n])
}

func (suite *ConnTestSuite) TestReadPassthrough() {
	data := []byte("raw bytes")

	suite.connMock.
		On("Read", mock.AnythingOfType("[]uint8")).
		Run(func(args mock.Arguments) {
			copy(args.Get(0).([]byte), data)
		}).
		Return(len(data), nil).
		Once()

	conn := New(suite.connMock, false, false)

	buf := make([]byte, 128)
	n, err := conn.Read(buf)

	suite.NoError(err)
	suite.Equal(data, buf[:n])
}

func (suite *ConnTestSuite) TestWritePassthrough() {
	data := []byte("outgoing data")

	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(len(data), nil).
		Once()

	conn := New(suite.connMock, false, false)

	n, err := conn.Write(data)

	suite.NoError(err)
	suite.Equal(len(data), n)
}

func (suite *ConnTestSuite) TestWriteTLSEnabled() {
	data := []byte("outgoing data")

	suite.connMock.
		On("Write", mock.AnythingOfType("[]uint8")).
		Return(len(data), nil).
		Once()

	conn := New(suite.connMock, false, true)

	n, err := conn.Write(data)

	suite.NoError(err)
	suite.Equal(len(data), n)
}

func TestConn(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConnTestSuite{})
}
