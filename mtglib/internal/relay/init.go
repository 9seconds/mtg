package relay

type Logger interface {
	Printf(msg string, args ...interface{})
}
