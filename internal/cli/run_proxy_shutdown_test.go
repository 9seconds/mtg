package cli

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// An ordinary stop closes the listener, which wakes Serve up with an accept
// error. That must not turn into exit code 1.
func TestWaitAndShutdownOrdinaryStopIsClean(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)

	cancel() // SIGTERM

	err := waitAndShutdown(ctx, serveErr, func() {
		serveErr <- errors.New("use of closed network connection")
	})
	require.NoError(t, err)
}

func TestWaitAndShutdownReportsServeFailure(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	serveErr := make(chan error, 1)
	stopped := false

	// Serve failed on its own: the error is sent first, then cancel().
	serveErr <- errors.New("too many open files")
	cancel()

	err := waitAndShutdown(ctx, serveErr, func() { stopped = true })
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many open files")
	assert.True(t, stopped)
}
