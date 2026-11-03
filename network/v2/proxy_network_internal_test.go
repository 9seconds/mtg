package network

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

// Regression test for https://github.com/9seconds/mtg/issues/439: the dialer
// used to reach a SOCKS upstream must not inherit the user-supplied DoT/DoH
// resolver, otherwise internal names (docker compose, k8s, /etc/hosts) fail
// to resolve through a public resolver that does not know them.
func TestProxyServerDialerDropsCustomResolver(t *testing.T) {
	customResolver := &net.Resolver{
		PreferGo: true,
		Dial: func(_ context.Context, _, _ string) (net.Conn, error) {
			return nil, errors.New("custom resolver must not be queried for SOCKS upstream")
		},
	}

	base := NewWithTCPNotSentLowat(
		customResolver,
		"mtg-test",
		7*time.Second,
		0,
		0,
		DefaultKeepAliveConfig,
		0,
	)

	if base.NativeDialer().Resolver != customResolver {
		t.Fatalf("base.NativeDialer().Resolver = %v, want custom resolver", base.NativeDialer().Resolver)
	}

	d := proxyServerDialer(base)

	if d.Resolver != nil {
		t.Errorf("proxyServerDialer().Resolver = %v, want nil (must use OS resolver)", d.Resolver)
	}

	if d.Timeout != base.NativeDialer().Timeout {
		t.Errorf("proxyServerDialer().Timeout = %v, want %v", d.Timeout, base.NativeDialer().Timeout)
	}

	if d.FallbackDelay != base.NativeDialer().FallbackDelay {
		t.Errorf("proxyServerDialer().FallbackDelay = %v, want %v", d.FallbackDelay, base.NativeDialer().FallbackDelay)
	}
}
