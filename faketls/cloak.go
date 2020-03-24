package faketls

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/9seconds/mtg/wrappers/rwc"
)

const (
	cloakLastActivityTimeout = 5 * time.Second
	cloakMaxTimeout          = 30 * time.Second
)

func cloak(one, another io.ReadWriteCloser) {
	defer func() {
		one.Close()
		another.Close()
	}()

	channelPing := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	one = rwc.NewPing(ctx, one, channelPing)
	another = rwc.NewPing(ctx, another, channelPing)
	wg := &sync.WaitGroup{}

	wg.Add(2)

	go cloakPipe(one, another, wg)

	go cloakPipe(another, one, wg)

	go func() {
		wg.Wait()
		cancel()
	}()

	go func() {
		lastActivityTimer := time.NewTimer(cloakLastActivityTimeout)
		defer lastActivityTimer.Stop()

		maxTimer := time.NewTimer(cloakMaxTimeout)
		defer maxTimer.Stop()

		for {
			select {
			case <-channelPing:
				lastActivityTimer.Stop()
				lastActivityTimer = time.NewTimer(cloakLastActivityTimeout)
			case <-ctx.Done():
				return
			case <-lastActivityTimer.C:
				cancel()
				return
			case <-maxTimer.C:
				cancel()
				return
			}
		}
	}()

	<-ctx.Done()
}

func cloakPipe(one io.Writer, another io.Reader, wg *sync.WaitGroup) {
	defer wg.Done()

	buf := acquireCloakBuffer()
	defer releaseCloakBuffer(buf)

	io.CopyBuffer(one, another, *buf) // nolint: errcheck
}
