package relay_test

type loggerMock struct{}

func (l loggerMock) Printf(format string, args ...interface{}) {}
