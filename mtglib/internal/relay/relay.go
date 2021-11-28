package relay

import (
	"context"
	"io"
	"sync"
)

func Relay(ctx context.Context, log Logger, telegramConn, clientConn io.ReadWriteCloser) {
	defer telegramConn.Close()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		telegramConn.Close()
		clientConn.Close()
	}()

	buffers := acquireEastWest()
	defer releaseEastWest(buffers)

	wg := &sync.WaitGroup{}
	wg.Add(2) // nolint: gomnd

	go pump(log, telegramConn, clientConn, wg, buffers.east, "east -> west")

	pump(log, clientConn, telegramConn, wg, buffers.west, "west -> east")

	wg.Wait()
}

func pump(log Logger, src io.ReadCloser, dst io.WriteCloser, wg *sync.WaitGroup,
	buf []byte, direction string) {
	defer wg.Done()
	defer src.Close()
	defer dst.Close()

	if n, err := io.CopyBuffer(dst, src, buf); err != nil {
		log.Printf("cannot pump %s (written %d bytes): %w", direction, n, err)
	}
}
