package utils

import "io"

const readFullBufferSize = 1024 + 1 // +1 because telegram opreates with blocks mod 4

func ReadFull(src io.Reader) ([]byte, error) {
	buf := make([]byte, readFullBufferSize)
	n := readFullBufferSize
	rv := []byte{}

	for n == len(buf) {
		n, err := src.Read(buf)
		if err != nil {
			return nil, err //nolint: wrapcheck
		}

		rv = append(rv, buf[:n]...)
	}

	return rv, nil
}
