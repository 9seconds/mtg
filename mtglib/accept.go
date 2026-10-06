package mtglib

import (
	"context"
	"errors"
	"net"
	"syscall"
	"time"
)

const (
	acceptRetryDelayMin = 5 * time.Millisecond
	acceptRetryDelayMax = time.Second
)

// IsTemporaryAcceptError reports whether an Accept error is transient: the
// listener is still usable and accepting should be retried after a pause.
// Running out of file descriptors under a connection flood (EMFILE/ENFILE) is
// the important case: returning from the accept loop on such an error leaves
// the process alive but deaf until it is restarted.
func IsTemporaryAcceptError(err error) bool {
	if err == nil || errors.Is(err, net.ErrClosed) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return errors.Is(err, syscall.EMFILE) ||
		errors.Is(err, syscall.ENFILE) ||
		errors.Is(err, syscall.ENOBUFS) ||
		errors.Is(err, syscall.ENOMEM) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.ECONNRESET)
}

// AcceptRetryDelay returns the pause before the next Accept attempt after a
// temporary error: it starts at 5ms and doubles up to 1s, like net/http.
func AcceptRetryDelay(previous time.Duration) time.Duration {
	if previous == 0 {
		return acceptRetryDelayMin
	}

	return min(previous*2, acceptRetryDelayMax)
}

// acceptWithRetry accepts the next connection, retrying temporary errors with
// a growing pause. It returns an error only when the listener is unusable or
// ctx is done.
func acceptWithRetry(ctx context.Context, listener net.Listener, logger Logger) (net.Conn, error) {
	var delay time.Duration

	for {
		conn, err := listener.Accept()
		if err == nil {
			return conn, nil
		}

		if ctx.Err() != nil || !IsTemporaryAcceptError(err) {
			return nil, err
		}

		delay = AcceptRetryDelay(delay)
		logger.BindStr("retry_in", delay.String()).WarningError("temporary accept error", err)

		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()

			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
