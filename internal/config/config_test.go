package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/9seconds/mtg/v2/internal/config"
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
	_, err := config.Parse([]byte{})
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParseBrokenToml() {
	_, err := config.Parse(suite.ReadConfig("broken.toml"))
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParseOnlySecret() {
	_, err := config.Parse(suite.ReadConfig("only_secret.toml"))
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParseMinimalConfig() {
	conf, err := config.Parse(suite.ReadConfig("minimal.toml"))
	suite.NoError(err)
	suite.Equal("7oe1GqLy6TBc38CV3jx7q09nb29nbGUuY29t", conf.Secret.Base64())
	suite.Equal("0.0.0.0:3128", conf.BindTo.String())
}

func (suite *ConfigTestSuite) TestParsePublicIP() {
	conf, err := config.Parse(suite.ReadConfig("public_ip.toml"))
	suite.NoError(err)
	suite.Equal("203.0.113.1", conf.PublicIPv4.Get(nil).String())
	suite.Equal("2001:db8::1", conf.PublicIPv6.Get(nil).String())
}

func (suite *ConfigTestSuite) TestParsePublicIPv4Only() {
	conf, err := config.Parse(suite.ReadConfig("public_ip_v4_only.toml"))
	suite.NoError(err)
	suite.Equal("203.0.113.1", conf.PublicIPv4.Get(nil).String())
	suite.Nil(conf.PublicIPv6.Get(nil))
}

func (suite *ConfigTestSuite) TestParsePublicIPInvalid() {
	_, err := config.Parse(suite.ReadConfig("public_ip_invalid.toml"))
	suite.Error(err)
}

func (suite *ConfigTestSuite) TestParsePublicIPNotSet() {
	conf, err := config.Parse(suite.ReadConfig("minimal.toml"))
	suite.NoError(err)
	suite.Nil(conf.PublicIPv4.Get(nil))
	suite.Nil(conf.PublicIPv6.Get(nil))
}

func (suite *ConfigTestSuite) TestString() {
	conf, err := config.Parse(suite.ReadConfig("minimal.toml"))
	suite.NoError(err)
	suite.NotEmpty(conf.String())
}

func TestConfig(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ConfigTestSuite{})
}
