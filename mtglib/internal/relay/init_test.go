package relay_test

import (
	"bytes"
	"io"
	"sync"
)

type loggerMock struct{}

func (l loggerMock) Printf(format string, args ...interface{}) {}

type rwcMock struct {
	bytes.Buffer

	closed bool
	mutex  sync.Mutex
}

func (r *rwcMock) Read(p []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return 0, io.EOF
	}

	return r.Buffer.Read(p)
}

func (r *rwcMock) Write(p []byte) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if r.closed {
		return 0, io.EOF
	}

	return r.Buffer.Write(p)
}

func (r *rwcMock) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.closed = true

	return nil
}
