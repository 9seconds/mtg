package tls

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite

	dst *bytes.Buffer
}

func (suite *UtilsTestSuite) SetupTest() {
	suite.dst = &bytes.Buffer{}
}

func (suite *UtilsTestSuite) TestReadRecord() {
	payload := []byte("hello world")
	raw := MakeTLSRecord(0x17, payload)

	recordType, length, err := ReadRecord(bytes.NewReader(raw), suite.dst)

	suite.NoError(err)
	suite.Equal(byte(0x17), recordType)
	suite.Equal(int64(len(payload)), length)
	suite.Equal(payload, suite.dst.Bytes())
}

func (suite *UtilsTestSuite) TestReadRecordChangeCipherSpec() {
	payload := []byte{1}
	raw := MakeTLSRecord(0x14, payload)

	recordType, length, err := ReadRecord(bytes.NewReader(raw), suite.dst)

	suite.NoError(err)
	suite.Equal(byte(0x14), recordType)
	suite.Equal(int64(1), length)
}

func (suite *UtilsTestSuite) TestReadRecordRejectsWrongVersion() {
	record := []byte{0x17, 3, 1, 0, 5, 0, 0, 0, 0, 0}

	_, _, err := ReadRecord(bytes.NewReader(record), suite.dst)
	suite.ErrorContains(err, "incorrect tls version")
}

func (suite *UtilsTestSuite) TestReadRecordEmptyReader() {
	_, _, err := ReadRecord(bytes.NewReader(nil), suite.dst)
	suite.Error(err)
}

func (suite *UtilsTestSuite) TestReadRecordTruncatedHeader() {
	_, _, err := ReadRecord(bytes.NewReader([]byte{0x17, 3}), suite.dst)
	suite.Error(err)
}

func (suite *UtilsTestSuite) TestReadRecordTruncatedPayload() {
	raw := MakeTLSRecord(0x17, []byte("full payload"))
	truncated := raw[:5+3]

	_, _, err := ReadRecord(bytes.NewReader(truncated), suite.dst)
	suite.Error(err)
}

func (suite *UtilsTestSuite) TestWriteRecord() {
	payload := []byte("hello world")

	err := WriteRecord(suite.dst, payload)
	suite.NoError(err)

	written := suite.dst.Bytes()
	suite.Equal(byte(0x17), written[0])
	suite.Equal([]byte{3, 3}, written[1:3])

	length := binary.BigEndian.Uint16(written[3:5])
	suite.Equal(uint16(len(payload)), length)
	suite.Equal(payload, written[5:])
}

func (suite *UtilsTestSuite) TestWriteRecordRoundTrip() {
	payload := []byte("round trip test")

	var wire bytes.Buffer

	err := WriteRecord(&wire, payload)
	suite.NoError(err)

	var recovered bytes.Buffer

	recordType, length, err := ReadRecord(&wire, &recovered)

	suite.NoError(err)
	suite.Equal(byte(0x17), recordType)
	suite.Equal(int64(len(payload)), length)
	suite.Equal(payload, recovered.Bytes())
}

func (suite *UtilsTestSuite) TestWriteRecordPropagatesError() {
	m := &WriterMock{}
	m.
		On("Write", mock.AnythingOfType("[]uint8")).
		Once().
		Return(0, errors.New("dist full"))

	err := WriteRecord(m, []byte("data"))
	suite.Error(err)

	m.AssertExpectations(suite.T())
}

func (suite *UtilsTestSuite) TestWriteRecordPayloadTooLarge() {
	err := WriteRecord(suite.dst, make([]byte, MaxRecordPayloadSize+1))
	suite.Error(err)
}

func TestUtils(t *testing.T) {
	t.Parallel()
	suite.Run(t, &UtilsTestSuite{})
}
