package cli_test

import (
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/internal/testlib"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/suite"
)

type GenerateSecretTestSuite struct {
	CommonTestSuite
}

func (suite *GenerateSecretTestSuite) SetupTest() {
	suite.CommonTestSuite.SetupTest()

	suite.cli.GenerateSecret.HostName = "google.com"
}

func (suite *GenerateSecretTestSuite) TestDefault() {
	output := testlib.CaptureStdout(func() {
		suite.NoError(suite.cli.GenerateSecret.Run(suite.cli, "dev"))
	})
	suite.True(strings.HasPrefix(output, "7"))

	secret, err := mtglib.ParseSecret(output)
	suite.NoError(err)
	suite.True(secret.Valid())
	suite.Equal("google.com", secret.Host)
}

func (suite *GenerateSecretTestSuite) TestHex() {
	suite.cli.GenerateSecret.Hex = true

	output := testlib.CaptureStdout(func() {
		suite.NoError(suite.cli.GenerateSecret.Run(suite.cli, "dev"))
	})
	suite.True(strings.HasPrefix(output, "ee"))

	secret, err := mtglib.ParseSecret(output)
	suite.NoError(err)
	suite.True(secret.Valid())
	suite.Equal("google.com", secret.Host)
}

func TestGenerateSecret(t *testing.T) {
	t.Parallel()
	suite.Run(t, &GenerateSecretTestSuite{})
}
