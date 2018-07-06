package utils

import "io"

const readCurrentDataBufferSize = 1024 + 1

func ReadCurrentData(src io.Reader) (rv []byte, err error) {
	buf := make([]byte, readCurrentDataBufferSize)
	n := readCurrentDataBufferSize

	for n == len(buf) {
		n, err = src.Read(buf)
		if err != nil {
			return nil, err
		}
		rv = append(rv, buf[:n]...)
	}

	return rv, nil
}
