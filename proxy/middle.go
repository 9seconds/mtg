package proxy

import (
	"sync"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/protocol"
	"github.com/9seconds/mtg/wrappers/packetack"
)

func middleConnection(request *protocol.TelegramRequest) error {
	telegramConn := packetack.NewProxy(request)
	defer telegramConn.Close()

	var clientConn conntypes.PacketAckFullReadWriteCloser
	switch request.ClientProtocol.ConnectionType() {
	case conntypes.ConnectionTypeAbridged:
		clientConn = packetack.NewClientAbridged(request.ClientConn)
	case conntypes.ConnectionTypeIntermediate:
		clientConn = packetack.NewClientIntermediate(request.ClientConn)
	case conntypes.ConnectionTypeSecure:
		clientConn = packetack.NewClientIntermediateSecure(request.ClientConn)
	default:
		panic("unknown connection type")
	}

	wg := &sync.WaitGroup{}
	wg.Add(2)

	go middlePipe(telegramConn, clientConn, wg, request.Logger)
	go middlePipe(clientConn, telegramConn, wg, request.Logger)

	wg.Wait()

	return nil
}

func middlePipe(dst conntypes.PacketAckWriteCloser,
	src conntypes.PacketAckReadCloser,
	wg *sync.WaitGroup,
	logger *zap.SugaredLogger) {
	defer func() {
		dst.Close()
		src.Close()
		wg.Done()
	}()

	for {
		acks := conntypes.ConnectionAcks{}
		packet, err := src.Read(&acks)
		if err != nil {
			logger.Debugw("Cannot read packet", "error", err)
			return
		}

		if err = dst.Write(packet, &acks); err != nil {
			logger.Debugw("Cannot send packet", "error", err)
			return
		}
	}
}
