package utils_test

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/internal/utils"
	"github.com/9seconds/mtg/v2/mtglib"
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

	log := &recordingLogger{}

	listener := utils.WrapListener(&firstConnBroken{Listener: base}, log)
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

	// The skipped connection must not vanish silently.
	if warnings := log.warnings(); len(warnings) != 1 {
		t.Fatalf("expected one warning about the skipped connection, got %v", warnings)
	}
}

// brokenConns returns only already closed TCP connections: setting socket
// options fails on every one of them.
type brokenConns struct {
	net.Listener

	calls atomic.Int32
}

func (l *brokenConns) Accept() (net.Conn, error) {
	l.calls.Add(1)

	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err //nolint: wrapcheck
	}

	conn.Close() //nolint: errcheck

	return conn, nil
}

// If setting socket options fails for every connection, the warning must be
// rate limited instead of written per connection.
func TestListenerRateLimitsSkipWarning(t *testing.T) {
	t.Parallel()

	base, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	log := &recordingLogger{}
	broken := &brokenConns{Listener: base}

	listener := utils.WrapListener(broken, log)

	const conns = 10

	var dialed sync.WaitGroup

	for range conns {
		dialed.Go(func() {
			conn, err := net.Dial("tcp", base.Addr().String())
			if err == nil {
				conn.Close() //nolint: errcheck
			}
		})
	}

	done := make(chan struct{})

	go func() {
		defer close(done)

		listener.Accept() //nolint: errcheck
	}()

	dialed.Wait()

	// The listener has skipped every connection once it asks for one more.
	deadline := time.Now().Add(2 * time.Second)

	for broken.calls.Load() <= conns && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}

	listener.Close() //nolint: errcheck
	<-done

	if got := broken.calls.Load(); got <= conns {
		t.Fatalf("expected %d skipped connections, got %d", conns, got-1)
	}

	if warnings := log.warnings(); len(warnings) != 1 {
		t.Fatalf("expected one rate-limited warning for %d skipped connections, got %v", conns, warnings)
	}
}

// recordingLogger records warnings.
type recordingLogger struct {
	mu   sync.Mutex
	msgs []string
}

func (l *recordingLogger) Named(_ string) mtglib.Logger          { return l }
func (l *recordingLogger) BindInt(_ string, _ int) mtglib.Logger { return l }
func (l *recordingLogger) BindStr(_, _ string) mtglib.Logger     { return l }
func (l *recordingLogger) BindJSON(_, _ string) mtglib.Logger    { return l }
func (l *recordingLogger) Printf(_ string, _ ...any)             {}
func (l *recordingLogger) Info(_ string)                         {}
func (l *recordingLogger) InfoError(_ string, _ error)           {}
func (l *recordingLogger) Warning(msg string)                    { l.record(msg) }
func (l *recordingLogger) Debug(_ string)                        {}
func (l *recordingLogger) DebugError(_ string, _ error)          {}

func (l *recordingLogger) WarningError(msg string, err error) {
	l.record(msg + ": " + err.Error())
}

func (l *recordingLogger) record(msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.msgs = append(l.msgs, msg)
}

func (l *recordingLogger) warnings() []string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return append([]string(nil), l.msgs...)
}
