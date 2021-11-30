package relay

import "time"

const (
	copyBufferSize   = 32 * 1024
	writerBufferSize = 2 * copyBufferSize
	readTimeout      = 10 * time.Millisecond
)

type Logger interface {
	Printf(msg string, args ...interface{})
}
