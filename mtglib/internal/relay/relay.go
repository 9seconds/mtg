package relay

import (
	"context"
	"errors"
	"io"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

// RelayResult holds byte counts from a relay session.
type RelayResult struct {
	ClientToTelegram int64
	TelegramToClient int64
}

func Relay(ctx context.Context, log Logger, telegramConn, clientConn essentials.Conn) RelayResult {
	defer telegramConn.Close() //nolint: errcheck
	defer clientConn.Close()   //nolint: errcheck

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctx.Done()
		telegramConn.Close() //nolint: errcheck
		clientConn.Close()   //nolint: errcheck
	}()

	var clientToTg int64

	closeChan := make(chan struct{})

	go func() {
		defer close(closeChan)

		clientToTg = pump(log, telegramConn, clientConn, "client -> telegram")
	}()

	tgToClient := pump(log, clientConn, telegramConn, "telegram -> client")

	<-closeChan

	return RelayResult{
		ClientToTelegram: clientToTg,
		TelegramToClient: tgToClient,
	}
}

func pump(log Logger, src, dst essentials.Conn, direction string) int64 {
	var buf [tls.MaxRecordPayloadSize]byte

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

	return n
}
