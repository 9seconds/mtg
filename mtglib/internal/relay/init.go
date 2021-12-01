package relay

import "time"

const (
	copyBufferSize   = 64 * 1024
	writerBufferSize = 128 * 1024
	readTimeout      = 10 * time.Millisecond
)

type Logger interface {
	Printf(msg string, args ...interface{})
}
