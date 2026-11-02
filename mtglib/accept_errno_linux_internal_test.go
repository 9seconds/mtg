package mtglib

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"syscall"
	"testing"
)

// On Linux accept(2) returns pending network errors of the new connection.
// They must not end the accept loop: the next connection is accepted at once.
func TestAcceptWithRetrySkipsAcceptedConnErrors(t *testing.T) {
	t.Parallel()

	for _, errno := range acceptedConnErrnos {
		wrapped := &net.OpError{Op: "accept", Err: os.NewSyscallError("accept4", errno)}
		listener := &scriptedListener{results: []error{wrapped, nil}}

		conn, err := acceptWithRetry(context.Background(), listener, NoopLogger{})
		if err != nil {
			t.Fatalf("%v: expected the next connection, got %v", errno, err)
		}

		conn.Close() //nolint: errcheck

		if listener.calls != 2 {
			t.Fatalf("%v: expected 2 Accept calls, got %d", errno, listener.calls)
		}
	}
}

func TestIsAcceptedConnError(t *testing.T) {
	t.Parallel()

	if !isAcceptedConnError(fmt.Errorf("accept: %w", syscall.EHOSTUNREACH)) {
		t.Error("EHOSTUNREACH must be an accepted connection error")
	}

	for _, err := range []error{nil, net.ErrClosed, syscall.EMFILE, syscall.EBADF, errors.New("boom")} {
		if isAcceptedConnError(err) {
			t.Errorf("%v must not be an accepted connection error", err)
		}
	}
}

// After shutdown the loop must stop even on these errors.
func TestAcceptWithRetryStopsOnShutdownWithAcceptedConnError(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := acceptWithRetry(ctx, &scriptedListener{results: []error{syscall.ENETUNREACH, nil}}, NoopLogger{})
	if err == nil {
		t.Fatal("expected an error after shutdown")
	}
}
