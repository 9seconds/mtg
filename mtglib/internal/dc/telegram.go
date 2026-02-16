package dc

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
)

type Telegram struct {
	logger   Logger
	lock     sync.RWMutex
	view     dcView
	preferIP preferIP
	client   *telegram.Client
}

func (t *Telegram) GetAddresses(dc int) []Addr {
	t.lock.RLock()
	defer t.lock.RUnlock()

	switch t.preferIP {
	case preferIPOnlyIPv4:
		return t.view.getV4(dc)
	case preferIPOnlyIPv6:
		return t.view.getV4(dc)
	case preferIPPreferIPv4:
		return append(t.view.getV4(dc), t.view.getV6(dc)...)
	}

	return append(t.view.getV6(dc), t.view.getV4(dc)...)
}

func (t *Telegram) Run(ctx context.Context, updateEach time.Duration) {
	if updateEach == 0 {
		updateEach = DefaultUpdateDCAddressesEach
	}

	t.update(ctx)

	ticker := time.NewTicker(updateEach)
	defer func() {
		ticker.Stop()

		select {
		case <-ctx.Done():
		case <-ticker.C:
		default:
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.update(ctx)
		}
	}
}

func (t *Telegram) update(ctx context.Context) {
	collected := dcAddrSet{
		v4: map[int][]Addr{},
		v6: map[int][]Addr{},
	}

	err := t.client.Run(ctx, func(tgctx context.Context) error {
		conf, err := t.client.API().HelpGetConfig(tgctx)
		if err != nil {
			return err
		}

		for _, opt := range conf.DCOptions {
			addr := net.JoinHostPort(opt.IPAddress, strconv.Itoa(opt.Port))

			if opt.Ipv6 {
				collected.v6[opt.ID] = append(collected.v6[opt.ID], Addr{
					Network: "tcp6",
					Address: addr,
				})
			} else {
				collected.v4[opt.ID] = append(collected.v4[opt.ID], Addr{
					Network: "tcp4",
					Address: addr,
				})
			}
		}

		return nil
	})
	if err != nil {
		t.logger.WarningError("update has failed", err)
		return
	}

	t.lock.Lock()
	t.view.collected = collected
	t.lock.Unlock()

	t.logger.Info(fmt.Sprintf("updated DC list: %v", collected))
}

func New(logger Logger, ipPreference string, userOverrides map[int][]string) (*Telegram, error) {
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

	overrides := dcAddrSet{}
	for dc, addrs := range userOverrides {
		for _, addr := range addrs {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				return nil, fmt.Errorf("incorrect host %s: %w", addr, err)
			}

			parsed := net.ParseIP(host)
			if parsed == nil {
				return nil, fmt.Errorf("incorrect host %s", addr)
			}

			if parsed.To4() != nil {
				overrides.v4[dc] = append(overrides.v4[dc], Addr{
					Network: "tcp4",
					Address: addr,
				})
			} else {
				overrides.v6[dc] = append(overrides.v6[dc], Addr{
					Network: "tcp6",
					Address: addr,
				})
			}
		}
	}

	return &Telegram{
		view: dcView{
			overrides: overrides,
		},
		logger:   logger,
		client:   telegram.NewClient(defaultAppID, defaultAppHash, telegram.Options{}),
		preferIP: pref,
	}, nil
}
