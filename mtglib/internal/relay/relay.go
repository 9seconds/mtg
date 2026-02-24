package relay

import (
	"context"
	"errors"
	"io"

	"github.com/9seconds/mtg/v2/essentials"
)

func Relay(ctx context.Context, log Logger, telegramConn, clientConn essentials.Conn) {
	defer telegramConn.Close() //nolint: errcheck
	defer clientConn.Close()   //nolint: errcheck

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		telegramConn.Close() //nolint: errcheck
		clientConn.Close()   //nolint: errcheck
	}()

	closeChan := make(chan struct{})

	go func() {
		defer close(closeChan)

		pump(log, telegramConn, clientConn, "client -> telegram")
	}()

	pump(log, clientConn, telegramConn, "telegram -> client")

	<-closeChan
}

func pump(log Logger, src, dst essentials.Conn, direction string) {
	var buf [copyBufferSize]byte

	defer src.CloseRead()  //nolint: errcheck
	defer dst.CloseWrite() //nolint: errcheck

	n, err := io.CopyBuffer(src, dst, buf[:])

	switch {
	case err == nil:
		log.Printf("%s has been finished", direction)
	case errors.Is(err, io.EOF):
		log.Printf("%s has been finished because of EOF. Written %d bytes", direction, n)
	default:
		log.Printf("%s has been finished (written %d bytes): %v", direction, n, err)
	}
}
