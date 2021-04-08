package mtglib

import "time"

type ProxyOpts struct {
	Secret             Secret
	Network            Network
	AntiReplayCache    AntiReplayCache
	TimeAttackDetector TimeAttackDetector
	IPBlocklist        IPBlocklist
	EventStream        EventStream
	Logger             Logger

	BufferSize         uint
	Concurrency        uint
	DomainFrontingPort uint
	IdleTimeout        time.Duration
	PreferIP           string
}

func (p ProxyOpts) valid() error {
	switch {
	case p.Network == nil:
		return ErrNetworkIsNotDefined
	case p.AntiReplayCache == nil:
		return ErrAntiReplayCacheIsNotDefined
	case p.IPBlocklist == nil:
		return ErrIPBlocklistIsNotDefined
	case p.EventStream == nil:
		return ErrEventStreamIsNotDefined
	case p.TimeAttackDetector == nil:
		return ErrTimeAttackDetectorIsNotDefined
	case p.Logger == nil:
		return ErrLoggerIsNotDefined
	case !p.Secret.Valid():
		return ErrSecretInvalid
	}

	return nil
}

func (p ProxyOpts) getBufferSize() int {
	if p.BufferSize < 1 {
		return DefaultBufferSize
	}

	return int(p.BufferSize)
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

func (p ProxyOpts) getIdleTimeout() time.Duration {
	if p.IdleTimeout == 0 {
		return DefaultIdleTimeout
	}

	return p.IdleTimeout
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
