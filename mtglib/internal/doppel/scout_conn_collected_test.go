package doppel

import (
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
	"github.com/stretchr/testify/suite"
)

type ScoutConnCollectedTestSuite struct {
	suite.Suite
}

func (suite *ScoutConnCollectedTestSuite) TestAddSingle() {
	collected := NewScoutConnCollected()
	collected.Add(tls.TypeApplicationData, 100)

	suite.Len(collected.data, 1)
	suite.Equal(byte(tls.TypeApplicationData), collected.data[0].recordType)
}

func (suite *ScoutConnCollectedTestSuite) TestAddTimestampsAreMonotonic() {
	collected := NewScoutConnCollected()

	collected.Add(tls.TypeApplicationData, 100)

	time.Sleep(time.Microsecond)
	collected.Add(tls.TypeApplicationData, 100)

	time.Sleep(time.Microsecond)
	collected.Add(tls.TypeApplicationData, 100)

	for i := 1; i < len(collected.data); i++ {
		suite.True(collected.data[i].timestamp.After(collected.data[i-1].timestamp))
	}
}

func TestScoutConnCollected(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ScoutConnCollectedTestSuite{})
}
