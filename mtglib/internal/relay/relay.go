package relay

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

var bufPool = sync.Pool{
	New: func() any {
		b := make([]byte, tls.MaxRecordPayloadSize)
		return &b
	},
}

func Relay(ctx context.Context, log Logger, telegramConn, clientConn essentials.Conn) {
	defer telegramConn.Close() //nolint: errcheck
	defer clientConn.Close()   //nolint: errcheck

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stop := context.AfterFunc(ctx, func() {
		telegramConn.Close() //nolint: errcheck
		clientConn.Close()   //nolint: errcheck
	})
	defer stop()

	closeChan := make(chan struct{})

	go func() {
		defer close(closeChan)

		pump(log, telegramConn, clientConn, "client -> telegram")
	}()

	pump(log, clientConn, telegramConn, "telegram -> client")

	<-closeChan
}

func pump(log Logger, src, dst essentials.Conn, direction string) {
	bp := bufPool.Get().(*[]byte)
	defer bufPool.Put(bp)

	defer src.CloseRead()  //nolint: errcheck
	defer dst.CloseWrite() //nolint: errcheck

	n, err := io.CopyBuffer(src, dst, *bp)

	switch {
	case err == nil:
		log.Printf("%s has been finished", direction)
	case errors.Is(err, io.EOF):
		log.Printf("%s has been finished because of EOF. Written %d bytes", direction, n)
	default:
		log.Printf("%s has been finished (written %d bytes): %v", direction, n, err)
	}
}
