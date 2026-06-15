//go:build !linux

package desync

import (
	"errors"
	"io"
)

func Start(_ int) (io.Closer, error) {
	return nil, errors.New("raw TCP desync is supported only on linux")
}
