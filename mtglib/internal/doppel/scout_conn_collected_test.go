package doppel

import (
	"sync"
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

	data, _ := collected.Snapshot()

	suite.Len(data, 1)
	suite.Equal(byte(tls.TypeApplicationData), data[0].recordType)
}

func (suite *ScoutConnCollectedTestSuite) TestAddTimestampsAreMonotonic() {
	collected := NewScoutConnCollected()

	collected.Add(tls.TypeApplicationData, 100)

	time.Sleep(time.Microsecond)
	collected.Add(tls.TypeApplicationData, 100)

	time.Sleep(time.Microsecond)
	collected.Add(tls.TypeApplicationData, 100)

	data, _ := collected.Snapshot()

	for i := 1; i < len(data); i++ {
		suite.True(data[i].timestamp.After(data[i-1].timestamp))
	}
}

func (suite *ScoutConnCollectedTestSuite) TestConcurrentAddSnapshot() {
	collected := NewScoutConnCollected()

	var wg sync.WaitGroup

	wg.Add(3)

	go func() {
		defer wg.Done()

		for i := 0; i < 1000; i++ {
			collected.Add(tls.TypeApplicationData, i)
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < 100; i++ {
			collected.MarkWrite()
		}
	}()

	go func() {
		defer wg.Done()

		for i := 0; i < 1000; i++ {
			// call Snapshot concurrently to exercise the lock under -race
			collected.Snapshot() //nolint:errcheck
		}
	}()

	wg.Wait()

	data, writeIndex := collected.Snapshot()
	suite.Len(data, 1000)
	suite.GreaterOrEqual(writeIndex, 0)
}

func TestScoutConnCollected(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ScoutConnCollectedTestSuite{})
}
