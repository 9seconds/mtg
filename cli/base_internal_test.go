package cli

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type BaseTestSuite struct {
	suite.Suite

	b base
}

func (suite *BaseTestSuite) SetupTest() {
	suite.b = base{}
}

func (suite *BaseTestSuite) TestReadConfigNok() {
	suite.b.ConfigPath = filepath.Join("testdata", "unknown")
	suite.Error(suite.b.ReadConfig("dev"))
}

func (suite *BaseTestSuite) TestReadConfig() {
	suite.b.ConfigPath = filepath.Join("testdata", "minimal.toml")
	suite.NoError(suite.b.ReadConfig("dev"))
}

func TestBase(t *testing.T) {
	t.Parallel()
	suite.Run(t, &BaseTestSuite{})
}
