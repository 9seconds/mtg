package config2_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config2"
	"github.com/stretchr/testify/suite"
)

type ConfigTestSuite struct {
    suite.Suite
}

func (suite *ConfigTestSuite) ReadConfig(filename string) []byte {
	data, err := os.ReadFile(filepath.Join("testdata", filename))
	suite.NoError(err)

	return data
}

func (suite *ConfigTestSuite) TestParseEmpty() {
	_, err := config2.Parse([]byte{})
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParseBrokenToml() {
	_, err := config2.Parse(suite.ReadConfig("broken.toml"))
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParseOnlySecret() {
	_, err := config2.Parse(suite.ReadConfig("only_secret.toml"))
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParseMinimalConfig() {
	conf, err := config2.Parse(suite.ReadConfig("minimal.toml"))
	suite.NoError(err)
	suite.Equal("7oe1GqLy6TBc38CV3jx7q09nb29nbGUuY29t", conf.Secret.Base64())
	suite.Equal("0.0.0.0:3128", conf.BindTo.String())
}

func (suite *ConfigTestSuite) TestString() {
	conf, err := config2.Parse(suite.ReadConfig("minimal.toml"))
	suite.NoError(err)
	suite.NotEmpty(conf.String())
}

func TestConfig(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConfigTestSuite{})
}
