package logger

type StdLikeLogger interface {
	Printf(format string, args ...interface{})
}
