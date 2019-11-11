package proxy

import (
	"sync"

	"go.uber.org/zap"

	"mtg/conntypes"
	"mtg/protocol"
	"mtg/wrappers/packetack"
)

func middleConnection(request *protocol.TelegramRequest) {
	telegramConn, err := packetack.NewProxy(request)
	if err != nil {
		request.Logger.Debugw("Cannot dial to Telegram", "error", err)
		return
	}
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
