package files

import (
	"context"
	"errors"
	"io"
)

// ErrBadHTTPClient is returned if given HTTP client is initialized
// incorrectly.
var ErrBadHTTPClient = errors.New("incorrect http client")

// File is an abstraction for a entity that can be opened in some context.
type File interface {
	// Open returns an readable entity for a file. It is important to not forget
	// to close it after the usage.
	Open(context.Context) (io.ReadCloser, error)

	// String returns a short text description for the file
	String() string
}
