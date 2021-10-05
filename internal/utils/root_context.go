//go:build !windows
// +build !windows

package utils

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

func RootContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for range sigChan {
			cancel()
		}
	}()

	return ctx
}
