package relay

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/9seconds/mtg/v2/essentials"
)

func Relay(ctx context.Context, log Logger, telegramConn, clientConn essentials.Conn) {
	defer telegramConn.Close()
	defer clientConn.Close()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		telegramConn.Close()
		clientConn.Close()
	}()

	wg := &sync.WaitGroup{}
	wg.Add(2) // nolint: gomnd

	go pump(log, telegramConn, clientConn, wg, "client -> telegram")

	pump(log, clientConn, telegramConn, wg, "telegram -> client")

	wg.Wait()
}

func pump(log Logger, src, dst essentials.Conn, wg *sync.WaitGroup, direction string) {
	syncer := acquireSyncPair(src, dst)

	defer func() {
		syncer.Flush()
		releaseSyncPair(syncer)
		src.CloseRead()  // nolint: errcheck
		dst.CloseWrite() // nolint: errcheck
		wg.Done()
	}()

	n, err := syncer.Sync()

	switch {
	case err == nil:
		log.Printf("%s has been finished", direction)
	case errors.Is(err, io.EOF):
		log.Printf("%s has been finished because of EOF. Written %d bytes", direction, n)
	default:
		log.Printf("%s has been finished (written %d bytes): %v", direction, n, err)
	}
}
