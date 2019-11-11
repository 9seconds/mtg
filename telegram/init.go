package telegram

import (
	"net"
	"sync"
	"time"
)

const telegramDialTimeout = 10 * time.Second

var (
	Direct Telegram
	Middle Telegram

	initOnce sync.Once
)

func Init() {
	initOnce.Do(func() {
		Direct = &directTelegram{
			baseTelegram: baseTelegram{
				dialer:      net.Dialer{Timeout: telegramDialTimeout},
				v4DefaultDC: directV4DefaultIdx,
				V6DefaultDC: directV6DefaultIdx,
				v4Addresses: directV4Addresses,
				v6Addresses: directV6Addresses,
			},
		}

		tg := &middleTelegram{
			baseTelegram: baseTelegram{
				dialer: net.Dialer{Timeout: telegramDialTimeout},
			},
		}
		if err := tg.update(); err != nil {
			panic(err)
		}
		go tg.backgroundUpdate()

		Middle = tg
	})
}
