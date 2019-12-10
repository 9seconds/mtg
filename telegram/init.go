package telegram

import (
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

const telegramDialTimeout = 10 * time.Second

var (
	Direct Telegram
	Middle Telegram

	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		logger := zap.S().Named("telegram")

		Direct = &directTelegram{
			baseTelegram: baseTelegram{
				dialer:      net.Dialer{Timeout: telegramDialTimeout},
				logger:      logger.Named("direct"),
				v4DefaultDC: directV4DefaultIdx,
				v6DefaultDC: directV6DefaultIdx,
				v4Addresses: directV4Addresses,
				v6Addresses: directV6Addresses,
			},
		}

		tg := &middleTelegram{
			baseTelegram: baseTelegram{
				dialer: net.Dialer{Timeout: telegramDialTimeout},
				logger: logger.Named("middle"),
			},
		}
		if err := tg.update(); err != nil {
			panic(err)
		}
		go tg.backgroundUpdate()

		Middle = tg
	})
}
