package doppel

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ScoutTestSuite struct {
	TLSServerTestSuite

	scout Scout
}

func (suite *ScoutTestSuite) SetupSuite() {
	suite.TLSServerTestSuite.SetupSuite()

	suite.scout = Scout{
		network: suite.network,
		urls:    suite.urls,
	}
}

func (suite *ScoutTestSuite) TestCollectResults() {
	durations, err := suite.scout.Learn(suite.ctx)
	suite.NoError(err)
	suite.Less(3, len(durations))
}

func (suite *ScoutTestSuite) TestCollectNothing() {
	suite.ctxCancel()

	_, err := suite.scout.Learn(suite.ctx)
	suite.Error(err)
}

func TestScout(t *testing.T) {
	suite.Run(t, &ScoutTestSuite{})
}
