package relay

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

type syncPair struct {
	writer  *bufio.Writer
	copyBuf []byte

	mutex  sync.Mutex
	reader net.Conn
}

func (s *syncPair) Sync() (int64, error) {
	return io.CopyBuffer(s, s, s.copyBuf) // nolint: wrapcheck
}

func (s *syncPair) Read(p []byte) (int, error) {
	n, err := s.readBlocking(p, false)

	// nothing has been delivered for readTimeout time. Let's flush.
	if errors.Is(err, os.ErrDeadlineExceeded) {
		if err := s.Flush(); err != nil {
			return 0, fmt.Errorf("cannot flush writer hand-side: %w", err)
		}

		return s.readBlocking(p, true)
	}

	return n, err
}

func (s *syncPair) Write(p []byte) (int, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	n, err := s.writer.Write(p) // nolint: wrapcheck

	// optimization for a case when we have a small package and want to avoid a
	// delay in readTimeout. In that case, we assume that peer has finished to
	// sent a data it wants to send so we can flush without waiting for anything
	// else.
	if err == nil && n < copyBufferSize {
		err = s.writer.Flush()
	}

	return n, err
}

func (s *syncPair) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.writer.Flush() // nolint: wrapcheck
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
