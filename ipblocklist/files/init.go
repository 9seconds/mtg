package files

import (
	"context"
	"errors"
	"io"
)

var ErrBadHTTPClient = errors.New("incorrect http client")

type File interface {
	Open(context.Context) (io.ReadCloser, error)
}
