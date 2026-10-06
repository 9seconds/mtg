package utils_test

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/utils"
)

// firstConnBroken returns an already closed TCP connection first (setting
// socket options on it fails) and then real connections.
type firstConnBroken struct {
	net.Listener

	done atomic.Bool
}

func (l *firstConnBroken) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	if l.done.CompareAndSwap(false, true) {
		conn.Close() //nolint: errcheck
	}

	return conn, nil
}

// One connection whose socket options cannot be set (for example, reset by
// the client right away) must be skipped, not stop the whole accept loop.
func TestListenerSkipsConnectionWithBrokenSocket(t *testing.T) {
	t.Parallel()

	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	listener := utils.Listener{Listener: &firstConnBroken{Listener: base}}
	defer listener.Close() //nolint: errcheck

	for range 2 {
		go func() {
			conn, err := net.Dial("tcp", base.Addr().String())
			if err == nil {
				time.Sleep(200 * time.Millisecond)
				conn.Close() //nolint: errcheck
			}
		}()
	}

	result := make(chan error, 1)

	go func() {
		conn, err := listener.Accept()
		if err == nil {
			conn.Close() //nolint: errcheck
		}

		result <- err
	}()

	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("expected the second connection, got error %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Accept did not return the next connection")
	}
}
