package relay

import (
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
	minDurationInSeconds := int(minDuration.Seconds())
	maxDurationInSeconds := int(maxDuration.Seconds())
	number := minDurationInSeconds + rand.Intn(maxDurationInSeconds-minDurationInSeconds)

	return time.Duration(number) * time.Second
}
