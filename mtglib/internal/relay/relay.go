package relay

import (
	"context"
	"errors"
	"io"

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

	closeChan := make(chan struct{})

	go func() {
		defer close(closeChan)

		pump(log, telegramConn, clientConn, "client -> telegram")
	}()

	pump(log, clientConn, telegramConn, "telegram -> client")

	<-closeChan
}

func pump(log Logger, src, dst essentials.Conn, direction string) {
	defer src.CloseRead()  //nolint: errcheck
	defer dst.CloseWrite() //nolint: errcheck

	copyBuffer := acquireCopyBuffer()
	defer releaseCopyBuffer(copyBuffer)

	n, err := io.CopyBuffer(src, dst, *copyBuffer)

	switch {
	case err == nil:
		log.Printf("%s has been finished", direction)
	case errors.Is(err, io.EOF):
		log.Printf("%s has been finished because of EOF. Written %d bytes", direction, n)
	default:
		log.Printf("%s has been finished (written %d bytes): %v", direction, n, err)
	}
}
