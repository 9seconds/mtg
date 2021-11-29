package files

import (
	"context"
	"io"
)

type File interface {
	Open(context.Context) (io.ReadCloser, error)
}
