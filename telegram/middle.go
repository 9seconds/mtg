package telegram

import (
	"fmt"
	"sync"
	"time"

	"github.com/9seconds/mtg/conntypes"
	"github.com/9seconds/mtg/telegram/api"
	"go.uber.org/zap"
)

const middleTelegramBackgroundUpdateEvery = time.Hour

type middleTelegram struct {
	baseTelegram

	mutex sync.RWMutex
}

func (m *middleTelegram) Secret() []byte {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.baseTelegram.Secret()
}

func (m *middleTelegram) update() error {
	secret, err := api.Secret()
	if err != nil {
		return fmt.Errorf("cannot fetch secret: %w", err)
	}

	v4Addresses, v4DefaultDC, err := api.AddressesV4()
	if err != nil {
		return fmt.Errorf("cannot fetch addresses for ipv4: %w", err)
	}

	v6Addresses, v6DefaultDC, err := api.AddressesV6()
	if err != nil {
		return fmt.Errorf("cannot fetch addresses for ipv6: %w", err)
	}

	m.mutex.Lock()
	m.secret = secret
	m.v4DefaultDC = v4DefaultDC
	m.v6DefaultDC = v6DefaultDC
	m.v4Addresses = v4Addresses
	m.v6Addresses = v6Addresses
	m.mutex.Unlock()

	return nil
}

func (m *middleTelegram) backgroundUpdate() {
	logger := zap.S().Named("telegram")

	for range time.Tick(middleTelegramBackgroundUpdateEvery) {
		if err := m.update(); err != nil {
			logger.Warnw("Cannot update Telegram proxies", "error", err)
		}
	}
}

func (m *middleTelegram) Dial(dc conntypes.DC,
	protocol conntypes.ConnectionProtocol) (conntypes.StreamReadWriteCloser, error) {
	if dc == 0 {
		dc = conntypes.DCDefaultIdx
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.baseTelegram.dial(dc, protocol)
}
