package utils_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/stretchr/testify/suite"
)

type MakeQRCodeURLTestSuite struct {
	suite.Suite
}

func (suite *MakeQRCodeURLTestSuite) TestSomeData() {
	value := utils.MakeQRCodeURL("some data")

	parsed, err := url.Parse(value)
	suite.NoError(err)

	suite.Equal("some data", parsed.Query().Get("data"))
	suite.Equal("svg", parsed.Query().Get("format"))
	suite.Equal("api.qrserver.com", strings.TrimPrefix(parsed.Host, "www."))
	suite.Equal("v1/create-qr-code", strings.Trim(parsed.Path, "/"))
}

func TestMakeQRCodeURL(t *testing.T) {
	t.Parallel()
	suite.Run(t, &MakeQRCodeURLTestSuite{})
}
