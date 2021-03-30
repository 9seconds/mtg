package record_test

import (
	"testing"

	"github.com/9seconds/mtg/v2/mtglib/internal/faketls/record"
	"github.com/stretchr/testify/suite"
)

type TypeTestSuite struct {
	suite.Suite
}

func (suite *TypeTestSuite) TestChangeCipherSpec() {
	suite.Contains(record.TypeChangeCipherSpec.String(), "changeCipher")
	suite.Contains(record.TypeChangeCipherSpec.String(), "0x14")
	suite.NoError(record.TypeChangeCipherSpec.Valid())
}

func (suite *TypeTestSuite) TestHandshake() {
	suite.Contains(record.TypeHandshake.String(), "handshake")
	suite.Contains(record.TypeHandshake.String(), "0x16")
	suite.NoError(record.TypeHandshake.Valid())
}

func (suite *TypeTestSuite) TestApplicationData() {
	suite.Contains(record.TypeApplicationData.String(), "applicationData")
	suite.Contains(record.TypeApplicationData.String(), "0x17")
	suite.NoError(record.TypeApplicationData.Valid())
}

func (suite *TypeTestSuite) TestUnknown() {
	value := record.Type(0x20)

	suite.Contains(value.String(), "unknown")
	suite.Contains(value.String(), "0x20")
	suite.Error(value.Valid())
}

type VersionTestSuite struct {
	suite.Suite
}

func (suite *VersionTestSuite) Test10() {
	suite.Equal("tls1.0", record.Version10.String())
	suite.NoError(record.Version10.Valid())
}

func (suite *VersionTestSuite) Test11() {
	suite.Equal("tls1.1", record.Version11.String())
	suite.NoError(record.Version11.Valid())
}

func (suite *VersionTestSuite) Test12() {
	suite.Equal("tls1.2", record.Version12.String())
	suite.NoError(record.Version12.Valid())
}

func (suite *VersionTestSuite) Test13() {
	suite.Equal("tls1.3", record.Version13.String())
	suite.NoError(record.Version13.Valid())
}

func (suite *VersionTestSuite) TestUnknown() {
	value := record.Version(900)

	suite.Equal("tls?(900)", value.String())
	suite.Error(value.Valid())
}

func TestType(t *testing.T) {
	t.Parallel()
	suite.Run(t, &TypeTestSuite{})
}

func TestVersion(t *testing.T) {
	t.Parallel()
	suite.Run(t, &VersionTestSuite{})
}
