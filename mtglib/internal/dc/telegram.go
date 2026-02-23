package dc

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

type Telegram struct {
	ctx      context.Context
	lock     sync.RWMutex
	view     dcView
	preferIP preferIP
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

func New(ipPreference string) (*Telegram, error) {
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

	return &Telegram{
		preferIP: pref,
	}, nil
}
