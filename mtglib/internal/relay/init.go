package relay

import "time"

const (
	ConnectionTimeToLiveMin = 2 * time.Minute
	ConnectionTimeToLiveMax = 10 * time.Minute
	TimeoutMin              = 20 * time.Second
	TimeoutMax              = time.Minute
)

type Logger interface {
	Printf(msg string, args ...interface{})
}
