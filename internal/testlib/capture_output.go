package testlib

import (
	"bytes"
	"io"
	"os"
	"strings"
)

func CaptureStdout(callback func()) string {
	return captureOutput(&os.Stdout, callback)
}

func CaptureStderr(callback func()) string {
	return captureOutput(&os.Stderr, callback)
}

func captureOutput(filefp **os.File, callback func()) string {
	oldFp := *filefp

	defer func() {
		*filefp = oldFp
	}()

	reader, writer, _ := os.Pipe()
	buf := &bytes.Buffer{}
	closeChan := make(chan bool)

	go func() {
		io.Copy(buf, reader) //nolint: errcheck
		close(closeChan)
	}()

	*filefp = writer

	callback()

	writer.Close()
	<-closeChan

	return strings.TrimSpace(buf.String())
}
