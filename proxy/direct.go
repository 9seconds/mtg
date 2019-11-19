package proxy

import (
	"io"
	"sync"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/obfuscated2"
	"github.com/9seconds/mtg/protocol"
)

const directPipeBufferSize = 1024 * 1024

func directConnection(request *protocol.TelegramRequest) error {
	telegramConnRaw, err := obfuscated2.TelegramProtocol(request)
	if err != nil {
		return err
	}

	telegramConn := telegramConnRaw.(conntypes.StreamReadWriteCloser)

	defer telegramConn.Close()

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go directPipe(telegramConn, request.ClientConn, wg, request.Logger)

	go directPipe(request.ClientConn, telegramConn, wg, request.Logger)

	wg.Wait()

	return nil
}

func directPipe(dst io.WriteCloser, src io.ReadCloser, wg *sync.WaitGroup, logger *zap.SugaredLogger) {
	defer func() {
		dst.Close()
		src.Close()
		wg.Done()
	}()

	buf := make([]byte, directPipeBufferSize)
	if _, err := io.CopyBuffer(dst, src, buf); err != nil {
		logger.Debugw("Cannot pump sockets", "error", err)
	}
}
