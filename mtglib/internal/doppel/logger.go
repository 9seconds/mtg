package doppel

type Logger interface {
	Info(msg string)
	WarningError(msg string, err error)
}
