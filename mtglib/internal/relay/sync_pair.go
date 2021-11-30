package relay

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

type syncPair struct {
	writer  *bufio.Writer
	copyBuf []byte

	reader net.Conn
}

func (s *syncPair) Sync() (int64, error) {
	return io.CopyBuffer(s, s, s.copyBuf) // nolint: wrapcheck
}

func (s *syncPair) Read(p []byte) (int, error) {
	n, err := s.readBlocking(p, false)

	if errors.Is(err, os.ErrDeadlineExceeded) {
		if err := s.writer.Flush(); err != nil {
			return 0, fmt.Errorf("cannot flush writer hand-side: %w", err)
		}

		return s.readBlocking(p, true)
	}

	return n, err
}

func (s *syncPair) Write(p []byte) (int, error) {
	return s.writer.Write(p) // nolint: wrapcheck
}

func (s *syncPair) readBlocking(p []byte, blocking bool) (int, error) {
	var deadline time.Time

	if !blocking {
		deadline = time.Now().Add(readTimeout)
	}

	if err := s.reader.SetReadDeadline(deadline); err != nil {
		return 0, fmt.Errorf("cannot set read deadline: %w", err)
	}

	return s.reader.Read(p) // nolint: wrapcheck
}
