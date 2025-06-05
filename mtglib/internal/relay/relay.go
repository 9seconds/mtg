package relay

import (
	"context"
	"errors"
	"io"

	"github.com/IceCodeNew/mtg/essentials"
)

func Relay(ctx context.Context, log Logger, telegramConn, clientConn essentials.Conn) {
	defer func() {
		if err := telegramConn.Close(); err != nil {
			log.Printf("error closing telegramConn: %v", err)
		}
		if err := clientConn.Close(); err != nil {
			log.Printf("error closing clientConn: %v", err)
		}
	}()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		if err := telegramConn.Close(); err != nil {
			log.Printf("error closing telegramConn: %v", err)
		}
		if err := clientConn.Close(); err != nil {
			log.Printf("error closing clientConn: %v", err)
		}
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
