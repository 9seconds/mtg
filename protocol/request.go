package protocol

import (
	"context"

	"github.com/9seconds/mtg/conntypes"
	"go.uber.org/zap"
)

type TelegramRequest struct {
	Logger         *zap.SugaredLogger
	ClientConn     conntypes.StreamReadWriteCloser
	ConnID         conntypes.ConnID
	Ctx            context.Context
	Cancel         context.CancelFunc
	ClientProtocol ClientProtocol
}
