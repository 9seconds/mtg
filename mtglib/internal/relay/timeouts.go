package relay

import (
	"math"
	"math/rand"
	"time"
)

func getConnectionTimeToLive() time.Duration {
	return getTime(ConnectionTimeToLiveMin, ConnectionTimeToLiveMax)
}

func getTimeout() time.Duration {
	return getTime(TimeoutMin, TimeoutMax)
}

func getTime(minDuration, maxDuration time.Duration) time.Duration {
	minDurationInSeconds := minDuration.Seconds()
	maxDurationInSeconds := maxDuration.Seconds()
	middle := minDurationInSeconds + (maxDurationInSeconds-minDurationInSeconds)/2 // nolint: gomnd

	number := minDurationInSeconds + rand.ExpFloat64()*middle
	number = math.Round(math.Min(maxDurationInSeconds, number))

	return time.Duration(number) * time.Second
}
