package relay

import (
	"context"
	"net"
	"sync"
)

func Relay(ctx context.Context, log Logger, telegramConn, clientConn net.Conn) {
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

func pump(log Logger, src, dst net.Conn, wg *sync.WaitGroup, direction string) {
	defer wg.Done()
	defer src.Close()
	defer dst.Close()

	syncer := acquireSyncPair(src, dst)
	defer releaseSyncPair(syncer)

	if n, err := syncer.Sync(); err != nil {
		log.Printf("cannot pump %s (written %d bytes): %w", direction, n, err)
	}
}
