package files

import (
	"context"
	"fmt"
	"io"
	"os"
)

type localFile struct {
	path string
}

func (l localFile) Open(ctx context.Context) (io.ReadCloser, error) {
	return os.Open(l.path) //nolint: wrapcheck
}

func (l localFile) String() string {
	return l.path
}

// NewLocal returns an openable File for a path on a local file system.
func NewLocal(path string) (File, error) {
	if stat, err := os.Stat(path); os.IsNotExist(err) || stat.IsDir() || stat.Mode().Perm()&0o400 == 0 {
		return nil, fmt.Errorf("%s is not a readable file", path)
	}

	return localFile{
		path: path,
	}, nil
}
