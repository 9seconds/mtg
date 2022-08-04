package mtglib

import "time"

// ProxyOpts is a structure with settings to mtg proxy.
//
// This is not required per se, but this is to shorten function signature and
// give an ability to conveniently provide default values.
type ProxyOpts struct {
	// Secret defines a secret which should be used by a proxy.
	//
	// This is a mandatory setting.
	Secret Secret

	// Network defines a network instance which should be used for all network
	// communications made by proxies.
	//
	// This is a mandatory setting.
	Network Network

	// AntiReplayCache defines an instance of antireplay cache.
	//
	// This is a mandatory setting.
	AntiReplayCache AntiReplayCache

	// IPBlocklist defines an instance of IP blocklist.
	//
	// This is a mandatory setting.
	IPBlocklist IPBlocklist

	// IPAllowlist defines a whitelist of IPs to allow to use proxy.
	//
	// This is an optional setting, ignored by default (no restrictions).
	IPAllowlist IPBlocklist

	// EventStream defines an instance of event stream.
	//
	// This ia a mandatory setting.
	EventStream EventStream

	// Logger defines an instance of the logger.
	//
	// This is a mandatory setting.
	Logger Logger

	// BufferSize is a size of the copy buffer in bytes.
	//
	// Please remember that we multiply this number in 2, because when we relay
	// between proxies, we have to create 2 intermediate buffers: to and from.
	//
	// This is an optional setting.
	//
	// Deprecated: this setting is no longer makes any effect.
	BufferSize uint

	// Concurrency is a size of the worker pool for connection management.
	//
	// If we have more connections than this number, they are going to be
	// rejected.
	//
	// This is an optional setting.
	Concurrency uint

	// IdleTimeout is a timeout for relay when we have to break a stream.
	//
	// This is a timeout for any activity. So, if we have any message which will
	// pass to either direction, a timer is reset. If we have no any reads or
	// writes for this timeout, a connection will be aborted.
	//
	// This is an optional setting.
	IdleTimeout time.Duration

	// TolerateTimeSkewness is a time boundary that defines a time range where
	// faketls timestamp is acceptable.
	//
	// This means that if if you got a timestamp X, now is Y, then if |X-Y| <
	// TolerateTimeSkewness, then you accept a packet.
	//
	// This is an optional setting.
	TolerateTimeSkewness time.Duration

	// PreferIP defines an IP connectivity preference. Valid values are:
	// 'prefer-ipv4', 'prefer-ipv6', 'only-ipv4', 'only-ipv6'.
	//
	// This is an optional setting.
	PreferIP string

	// DomainFrontingPort is a port we use to connect to a fronting domain.
	//
	// This is required because secret does not specify a port. It specifies a
	// hostname only.
	//
	// This is an optional setting.
	DomainFrontingPort uint

	// AllowFallbackOnUnknownDC defines how proxy behaves if unknown DC was
	// requested. If this setting is set to false, then such connection will be
	// rejected. Otherwise, proxy will chose any DC.
	//
	// Telegram is designed in a way that any DC can serve any request, the
	// problem is a latency.
	//
	// This is an optional setting.
	AllowFallbackOnUnknownDC bool

	// UseTestDCs defines if we have to connect to production or to staging DCs of
	// Telegram.
	//
	// This is required if you use mtglib as an integration library for your
	// Telegram-related projects.
	//
	// This is an optional setting.
	UseTestDCs bool
}

func (p ProxyOpts) valid() error {
	switch {
	case p.Network == nil:
		return ErrNetworkIsNotDefined
	case p.AntiReplayCache == nil:
		return ErrAntiReplayCacheIsNotDefined
	case p.IPBlocklist == nil:
		return ErrIPBlocklistIsNotDefined
	case p.IPAllowlist == nil:
		return ErrIPAllowlistIsNotDefined
	case p.EventStream == nil:
		return ErrEventStreamIsNotDefined
	case p.Logger == nil:
		return ErrLoggerIsNotDefined
	case !p.Secret.Valid():
		return ErrSecretInvalid
	}

	return nil
}

func (p ProxyOpts) getConcurrency() int {
	if p.Concurrency == 0 {
		return DefaultConcurrency
	}

	return int(p.Concurrency)
}

func (p ProxyOpts) getDomainFrontingPort() int {
	if p.DomainFrontingPort == 0 {
		return DefaultDomainFrontingPort
	}

	return int(p.DomainFrontingPort)
}

func (p ProxyOpts) getTolerateTimeSkewness() time.Duration {
	if p.TolerateTimeSkewness == 0 {
		return DefaultTolerateTimeSkewness
	}

	return p.TolerateTimeSkewness
}

func (p ProxyOpts) getPreferIP() string {
	if p.PreferIP == "" {
		return DefaultPreferIP
	}

	return p.PreferIP
}

func (p ProxyOpts) getLogger(name string) Logger {
	return p.Logger.Named(name)
}
