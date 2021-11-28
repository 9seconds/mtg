package relay

const (
	bufferSize = 32 * 1024
)

type Logger interface {
	Printf(msg string, args ...interface{})
}
