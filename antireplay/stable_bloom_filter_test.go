package antireplay_test

import (
	"testing"

	"github.com/9seconds/mtg/v2/antireplay"
	"github.com/stretchr/testify/suite"
)

type StableBloomFilterTestSuite struct {
	suite.Suite
}

func (suite *StableBloomFilterTestSuite) TestOp() {
	filter := antireplay.NewStableBloomFilter(500, 0.001)

	suite.False(filter.SeenBefore([]byte{1, 2, 3}))
	suite.False(filter.SeenBefore([]byte{4, 5, 6}))
	suite.True(filter.SeenBefore([]byte{1, 2, 3}))
	suite.True(filter.SeenBefore([]byte{4, 5, 6}))
}

func TestStableBloomFilter(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StableBloomFilterTestSuite{})
}
