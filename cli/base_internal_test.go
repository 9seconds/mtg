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
	suite.Error(suite.b.ReadConfig(filepath.Join("testdata", "unknown"), "dev"))
}

func (suite *BaseTestSuite) TestReadConfig() {
	suite.NoError(suite.b.ReadConfig(filepath.Join("testdata", "minimal.toml"), "dev"))
}

func TestBase(t *testing.T) {
	t.Parallel()
	suite.Run(t, &BaseTestSuite{})
}
