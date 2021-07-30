package utils_test

import (
	"path/filepath"
	"testing"

	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/stretchr/testify/suite"
)

type ReadConfigTestSuite struct {
	suite.Suite
}

func (suite *ReadConfigTestSuite) GetConfigPath(filename string) string {
	return filepath.Join("testdata", filename)
}

func (suite *ReadConfigTestSuite) TestReadMinimal() {
	conf, err := utils.ReadConfig(suite.GetConfigPath("minimal.toml"))
	suite.NoError(err)
	suite.NoError(conf.Validate())
	suite.Equal("0.0.0.0:80", conf.BindTo.Get(""))
	suite.Equal("7mqFMMq3P2Tvvt_rPx5qhmFnb29nbGUuY29t", conf.Secret.Base64())
}

func (suite *ReadConfigTestSuite) TestReadAbsentFile() {
	_, err := utils.ReadConfig(suite.GetConfigPath("unknown.file"))
	suite.Error(err)
}

func (suite *ReadConfigTestSuite) TestBrokenFile() {
	_, err := utils.ReadConfig(suite.GetConfigPath("broken.toml"))
	suite.Error(err)
}

func (suite *ReadConfigTestSuite) TestMissedBindTo() {
	_, err := utils.ReadConfig(suite.GetConfigPath("missed-bindto.toml"))
	suite.Error(err)
}

func (suite *ReadConfigTestSuite) TestMissedSecret() {
	_, err := utils.ReadConfig(suite.GetConfigPath("missed-secret.toml"))
	suite.Error(err)
}

func (suite *ReadConfigTestSuite) TestEmpty() {
	_, err := utils.ReadConfig(suite.GetConfigPath("empty.toml"))
	suite.Error(err)
}

func TestReadConfig(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ReadConfigTestSuite{})
}
