//go:build linux
// +build linux

package relay

import "io"

func ioCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src) //nolint: wrapcheck
}
