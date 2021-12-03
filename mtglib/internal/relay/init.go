package relay

const (
	copyBufferSize = 64 * 1024
)

type Logger interface {
	Printf(msg string, args ...interface{})
}
