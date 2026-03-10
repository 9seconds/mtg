package doppel

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type ClockTestSuite struct {
	suite.Suite

	clock     Clock
	wg        sync.WaitGroup
	ctx       context.Context
	ctxCancel context.CancelFunc
}

func (suite *ClockTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())

	suite.ctx = ctx
	suite.ctxCancel = cancel
	suite.clock = Clock{
		stats: &Stats{
			k:      StatsDefaultK,
			lambda: StatsDefaultLambda,
		},
		tick: make(chan struct{}),
	}

	suite.wg.Go(func() {
		suite.clock.Start(suite.ctx)
	})
}

func (suite *ClockTestSuite) TearDownTest() {
	suite.ctxCancel()
	suite.wg.Wait()
}

func (suite *ClockTestSuite) TestTicks() {
	received := 0

	for range 3 {
		select {
		case <-suite.clock.tick:
			received++
		case <-time.After(2 * time.Second):
			suite.Fail("timed out waiting for tick")
		}
	}

	suite.Equal(3, received)
}

func (suite *ClockTestSuite) TestStopsOnCancel() {
	select {
	case <-suite.clock.tick:
	case <-time.After(2 * time.Second):
		suite.Fail("timed out waiting for first tick")
	}

	suite.ctxCancel()

	time.Sleep(50 * time.Millisecond)

	select {
	case <-suite.clock.tick:
		suite.Fail("received tick after cancel")
	default:
	}
}

func TestClock(t *testing.T) {
	t.Parallel()
	suite.Run(t, &ClockTestSuite{})
}
