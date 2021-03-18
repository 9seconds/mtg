package timeattack_test

import (
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/timeattack"
	"github.com/stretchr/testify/suite"
)

type DetectorTestSuite struct {
	suite.Suite
}

func (suite *DetectorTestSuite) TestOp() {
	d := timeattack.NewDetector(time.Second)

	suite.NoError(d.Valid(time.Now()))
	suite.NoError(d.Valid(time.Now().Add(100 * time.Millisecond)))
	suite.NoError(d.Valid(time.Now().Add(-100 * time.Millisecond)))
	suite.Error(d.Valid(time.Now().Add(time.Hour)))
	suite.Error(d.Valid(time.Now().Add(-time.Hour)))
}

func TestDetector(t *testing.T) {
	t.Parallel()
	suite.Run(t, &DetectorTestSuite{})
}
