package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/gotd/td/telegram"
)

type Telegram struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	lock      sync.RWMutex

	dialer    Dialer
	preferIP  preferIP
	addresses dcAddresses
	rpc       rpcClient
}

func (t *Telegram) Dial(ctx context.Context, dc int) (essentials.Conn, error) {
	var addresses []tgAddr

	t.lock.RLock()
	switch t.preferIP {
	case preferIPOnlyIPv4:
		addresses = t.addresses.getV4(dc)
	case preferIPOnlyIPv6:
		addresses = t.addresses.getV6(dc)
	case preferIPPreferIPv4:
		addresses = append(t.addresses.getV4(dc), t.addresses.getV6(dc)...)
	case preferIPPreferIPv6:
		addresses = append(t.addresses.getV6(dc), t.addresses.getV4(dc)...)
	}
	t.lock.RUnlock()

	var conn essentials.Conn

	err := errNoAddresses

	for _, v := range addresses {
		conn, err = t.dialer.DialContext(ctx, v.network, v.address)
		if err == nil {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("cannot dial to %d dc: %w", dc, err)
}

func (t *Telegram) IsKnownDC(dc int) bool {
	return t.addresses.isValidDC(dc)
}

func (t *Telegram) GetFallbackDC() int {
	return defaultDC
}

func (t *Telegram) Shutdown() {
	t.ctxCancel()
}

func (t *Telegram) Run(logger loggerInterface, updateEach time.Duration) {
	if updateEach == 0 {
		updateEach = defaultUpdateDCAddressesEach
	}

	t.update(logger)

	ticker := time.NewTicker(updateEach)
	defer func() {
		ticker.Stop()

		select {
		case <-ticker.C:
		default:
		}
	}()

	for {
		select {
		case <-t.ctx.Done():
			return
		case <-ticker.C:
			t.update(logger)
		}
	}
}

func (t *Telegram) update(logger loggerInterface) {
	otherAddresses, err := t.rpc.getDCAddresses(logger, t.ctx)
	if err != nil {
		logger.WarningError("Cannot update DC list", err)
		return
	}

	t.lock.Lock()
	t.addresses = otherAddresses
	t.lock.Unlock()

	logger.Info(fmt.Sprintf("DC are updated: %v", t.addresses))
}

func New(dialer Dialer, ipPreference string) (*Telegram, error) {
	var pref preferIP

	switch strings.ToLower(ipPreference) {
	case "prefer-ipv4":
		pref = preferIPPreferIPv4
	case "prefer-ipv6":
		pref = preferIPPreferIPv6
	case "only-ipv4":
		pref = preferIPOnlyIPv4
	case "only-ipv6":
		pref = preferIPOnlyIPv6
	default:
		return nil, fmt.Errorf("unknown ip preference %s", ipPreference)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Telegram{
		ctx:       ctx,
		ctxCancel: cancel,
		dialer:    dialer,
		preferIP:  pref,
		addresses: dcAddresses{
			v4: defaultV4Addresses,
			v6: defaultV6Addresses,
		},
		rpc: rpcClient{
			Client: telegram.NewClient(defaultAppID, defaultAppHash, telegram.Options{}),
		},
	}, nil
}
