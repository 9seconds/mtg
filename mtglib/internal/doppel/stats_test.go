package doppel

import (
	"math"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type StatsTestSuite struct {
	suite.Suite
}

func (suite *StatsTestSuite) GenWeibull(k, lambda float64, n int, seed uint64) []time.Duration {
	rng := rand.New(rand.NewPCG(seed, 0))
	samples := make([]time.Duration, n)

	for i := range samples {
		u := 1.0 - rng.Float64()
		ms := lambda * math.Pow(-math.Log(u), 1.0/k)
		d := time.Duration(ms * float64(time.Millisecond))

		if d < time.Microsecond {
			time.Sleep(time.Microsecond)
			d = time.Microsecond
		}

		samples[i] = d
	}

	return samples
}

func (suite *StatsTestSuite) TestNewStatsRecoverParameters() {
	knownK := 1.5
	knownLambda := 100.0

	samples := suite.GenWeibull(knownK, knownLambda, 5000, 42)
	stats := NewStats(samples, true)

	suite.InDelta(knownK, stats.k, 0.1)
	suite.InDelta(knownLambda, stats.lambda, 5.0)
}

func (suite *StatsTestSuite) TestNewStatsExponentialCase() {
	// When k=1, Weibull reduces to exponential distribution.
	knownK := 1.0
	knownLambda := 50.0

	samples := suite.GenWeibull(knownK, knownLambda, 5000, 123)
	stats := NewStats(samples, true)

	suite.InDelta(knownK, stats.k, 0.1)
	suite.InDelta(knownLambda, stats.lambda, 5.0)
}

func (suite *StatsTestSuite) TestNewStatsSmallK() {
	// k < 1 produces a heavy-tailed distribution typical for network delays.
	// Lambda must be large enough so samples stay above microsecond precision
	// after time.Duration round-trip.
	knownK := 0.6
	knownLambda := 100.0

	samples := suite.GenWeibull(knownK, knownLambda, 10000, 99)
	stats := NewStats(samples, true)

	suite.InDelta(knownK, stats.k, 0.05)
	suite.InDelta(knownLambda, stats.lambda, 5.0)
}

func (suite *StatsTestSuite) TestNewStatsLargeK() {
	// k > 1: light tail, concentrated around the mode.
	knownK := 5.0
	knownLambda := 200.0

	samples := suite.GenWeibull(knownK, knownLambda, 5000, 77)
	stats := NewStats(samples, true)

	suite.InDelta(knownK, stats.k, 0.3)
	suite.InDelta(knownLambda, stats.lambda, 5.0)
}

func (suite *StatsTestSuite) TestDelayNonNegative() {
	stats := &Stats{
		k:      1.5,
		lambda: 100.0,
	}

	for range 200 {
		dur := stats.Delay()
		suite.GreaterOrEqual(dur, time.Duration(0))
	}
}

func (suite *StatsTestSuite) TestDelayDistributionMean() {
	// Weibull mean = λ · Γ(1 + 1/k)
	k := 2.0
	lambda := 50.0
	stats := &Stats{k: k, lambda: lambda}

	n := 50000
	sum := 0.0

	for range n {
		dur := stats.Delay()
		sum += float64(dur) / float64(time.Millisecond)
	}

	sampleMean := sum / float64(n)
	expectedMean := lambda * math.Gamma(1.0+1.0/k)

	suite.InDelta(expectedMean, sampleMean, expectedMean*0.05)
}

func (suite *StatsTestSuite) TestNewStatsRoundTrip() {
	// Estimate parameters from data, then verify that Delay samples
	// from the fitted distribution have approximately the same mean.
	knownK := 1.2
	knownLambda := 80.0

	samples := suite.GenWeibull(knownK, knownLambda, 5000, 555)
	stats := NewStats(samples, true)

	n := 50000
	sum := 0.0

	for range n {
		dur := stats.Delay()
		sum += float64(dur) / float64(time.Millisecond)
	}

	sampleMean := sum / float64(n)
	expectedMean := knownLambda * math.Gamma(1.0+1.0/knownK)

	suite.InDelta(expectedMean, sampleMean, expectedMean*0.05)
}

func (suite *StatsTestSuite) TestSizeStartPhase() {
	stats := &Stats{k: 1.0, lambda: 1.0, drs: true}

	for range TLSCounterAccelAfter {
		size := stats.Size()
		suite.GreaterOrEqual(size, TLSRecordSizeStart-DRSNoise)
		suite.LessOrEqual(size, TLSRecordSizeStart)
	}
}

func (suite *StatsTestSuite) TestSizeAccelPhase() {
	stats := &Stats{k: 1.0, lambda: 1.0, drs: true}

	for range TLSCounterAccelAfter {
		stats.Size()
	}

	for range TLSCounterMaxAfter - TLSCounterAccelAfter {
		size := stats.Size()
		suite.GreaterOrEqual(size, TLSRecordSizeAccel-DRSNoise)
		suite.LessOrEqual(size, TLSRecordSizeAccel)
	}
}

func (suite *StatsTestSuite) TestSizeMaxPhase() {
	stats := &Stats{k: 1.0, lambda: 1.0, drs: true}

	for range TLSCounterMaxAfter {
		stats.Size()
	}

	for range 20 {
		size := stats.Size()
		suite.Equal(TLSRecordSizeMax, size)
	}
}

func (suite *StatsTestSuite) TestSizeResetsAfterInactivity() {
	stats := &Stats{k: 1.0, lambda: 1.0, drs: true}

	// Advance past start phase.
	for range TLSCounterMaxAfter {
		stats.Size()
	}

	suite.Equal(TLSRecordSizeMax, stats.Size())

	// Simulate inactivity by backdating sizeLastRequested.
	stats.sizeLastRequested = time.Now().Add(-TLSRecordSizeResetAfter - time.Millisecond)

	size := stats.Size()
	suite.GreaterOrEqual(size, TLSRecordSizeStart-DRSNoise)
	suite.LessOrEqual(size, TLSRecordSizeStart)
}

func (suite *StatsTestSuite) TestSizeNoDRSAlwaysMax() {
	stats := &Stats{k: 1.0, lambda: 1.0, drs: false}

	for range TLSCounterMaxAfter + 20 {
		suite.Equal(TLSRecordSizeMax, stats.Size())
	}
}

func (suite *StatsTestSuite) TestSizeNoDRSIgnoresCounter() {
	stats := &Stats{k: 1.0, lambda: 1.0, drs: false}

	// Even after many calls, always returns max.
	for range 200 {
		suite.Equal(TLSRecordSizeMax, stats.Size())
	}

	// Inactivity has no effect either.
	stats.sizeLastRequested = time.Now().Add(-TLSRecordSizeResetAfter - time.Millisecond)
	suite.Equal(TLSRecordSizeMax, stats.Size())
}

func TestStats(t *testing.T) {
	t.Parallel()
	suite.Run(t, &StatsTestSuite{})
}
