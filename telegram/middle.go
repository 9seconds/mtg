package telegram

import (
	"net"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/mtproto"
	"github.com/9seconds/mtg/wrappers"
)

type middleTelegram struct {
	middleTelegramCaller
}

func NewMiddleTelegram(conf *config.Config, logger *zap.SugaredLogger) Telegram {
	tg := &middleTelegram{
		middleTelegramCaller: middleTelegramCaller{
			baseTelegram: baseTelegram{
				dialer: tgDialer{net.Dialer{Timeout: telegramDialTimeout}},
			},
			logger: logger,
			httpClient: &http.Client{
				Timeout: middleTelegramHTTPClientTimeout,
			},
			dialerMutex: &sync.RWMutex{},
		},
	}

	if err := tg.update(); err != nil {
		panic(err)
	}
	go tg.autoUpdate()

	return tg
}

func (t *middleTelegram) Init(connOpts *mtproto.ConnectionOpts, conn wrappers.ReadWriteCloserWithAddr) (wrappers.ReadWriteCloserWithAddr, error) {
	return nil, nil
}
