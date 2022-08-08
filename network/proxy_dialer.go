package network

import (
	"net/url"
	"strconv"
	"time"
)

func newProxyDialer(baseDialer Dialer, proxyURL *url.URL) Dialer {
	params := proxyURL.Query()

	var (
		openThreshold        uint32 = ProxyDialerOpenThreshold
		halfOpenTimeout             = ProxyDialerHalfOpenTimeout
		resetFailuresTimeout        = ProxyDialerResetFailuresTimeout
	)

	if param := params.Get("open_threshold"); param != "" {
		if intNum, err := strconv.ParseUint(param, 10, 32); err == nil { //nolint: gomnd
			openThreshold = uint32(intNum)
		}
	}

	if param := params.Get("half_open_timeout"); param != "" {
		if dur, err := time.ParseDuration(param); err == nil && dur > 0 {
			halfOpenTimeout = dur
		}
	}

	if param := params.Get("reset_failures_timeout"); param != "" {
		if dur, err := time.ParseDuration(param); err == nil && dur > 0 {
			resetFailuresTimeout = dur
		}
	}

	return newCircuitBreakerDialer(baseDialer, openThreshold, halfOpenTimeout, resetFailuresTimeout)
}
