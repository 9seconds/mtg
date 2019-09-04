package protocol

import (
	"context"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/wrappers"
)

type TelegramRequest struct {
	Logger         *zap.SugaredLogger
	ClientConn     wrappers.StreamReadWriteCloser
	ConnID         wrappers.ConnID
	Ctx            context.Context
	Cancel         context.CancelFunc
	ClientProtocol ClientProtocol
}
