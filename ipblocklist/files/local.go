package files

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type localFile struct {
	root fs.FS
	name string
}

func (l localFile) Open(ctx context.Context) (io.ReadCloser, error) {
	return l.root.Open(l.name)
}

func NewLocal(path string) (File, error) {
	if stat, err := os.Stat(path); os.IsNotExist(err) || stat.IsDir() || stat.Mode().Perm()&0o400 == 0 {
		return nil, fmt.Errorf("%s is not a readable file", path)
	}

	return localFile{
		root: os.DirFS(filepath.Dir(path)),
		name: filepath.Base(path),
	}, nil
}
