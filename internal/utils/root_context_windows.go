//go:build windows
// +build windows

package utils

import (
	"context"
	"os"
	"os/signal"
)

func RootContext() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt)
	go func() {
		for range sigChan {
			cancel()
		}
	}()

	return ctx
}
