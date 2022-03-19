package proxy

import (
	"io"
	"sync"

	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/protocol"
	"go.uber.org/zap"
)

const directPipeBufferSize = 1024

func directConnection(request *protocol.TelegramRequest) error {
	telegramConnRaw, err := obfuscated2.TelegramProtocol(request)
	if err != nil {
		return err // nolint: wrapcheck
	}

	defer telegramConnRaw.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2) // nolint: gomnd

	go directPipe(telegramConnRaw, request.ClientConn, wg, request.Logger)

	go directPipe(request.ClientConn, telegramConnRaw, wg, request.Logger)

	wg.Wait()

	return nil
}

func directPipe(dst io.WriteCloser, src io.ReadCloser, wg *sync.WaitGroup, logger *zap.SugaredLogger) {
	defer func() {
		dst.Close()
		src.Close()
		wg.Done()
	}()

	buf := [directPipeBufferSize]byte{}

	if _, err := io.CopyBuffer(dst, src, buf[:]); err != nil {
		logger.Debugw("Cannot pump sockets", "error", err)
	}
}
