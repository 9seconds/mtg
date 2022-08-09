//go:build !linux
// +build !linux

package relay

import "io"

type writerOnly struct {
	io.Writer
}

func ioCopy(dst io.Writer, src io.Reader) (int64, error) {
	copyBuffer := acquireCopyBuffer()
	defer releaseCopyBuffer(copyBuffer)

	return io.CopyBuffer(writerOnly{dst}, src, *copyBuffer) //nolint: wrapcheck
}
