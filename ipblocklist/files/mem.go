package files

import (
	"context"
	"io"
	"net"
	"strings"
)

type memFile struct {
	data string
}

func (m memFile) Open(ctx context.Context) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(m.data)), nil
}

func (m memFile) String() string {
	return "mem"
}

// NewMem returns an openable file that is kept in RAM.
func NewMem(networks []*net.IPNet) File {
	builder := strings.Builder{}

	if len(networks) > 0 {
		builder.WriteString(networks[0].String())
	}

	for i := 1; i < len(networks); i++ {
		builder.WriteString("\n")
		builder.WriteString(networks[i].String())
	}

	return memFile{
		data: builder.String(),
	}
}
