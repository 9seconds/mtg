package dcprobe_test

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/9seconds/mtg/v2/mtglib/dcprobe"
)

// TestProbeAgainstTelegramDCs makes outbound TCP connections to public
// Telegram DCs. Skipped by default; opt-in with MTG_PROBE_NETWORK=1.
func TestProbeAgainstTelegramDCs(t *testing.T) {
	if os.Getenv("MTG_PROBE_NETWORK") != "1" {
		t.Skip("skipping network probe (set MTG_PROBE_NETWORK=1 to enable)")
	}

	cases := []struct {
		dc   int
		addr string
	}{
		{1, "149.154.175.50:443"},
		{2, "149.154.167.51:443"},
		{2, "95.161.76.100:443"},
		{3, "149.154.175.100:443"},
		{4, "149.154.167.91:443"},
		{5, "149.154.171.5:443"},
		{1, "[2001:b28:f23d:f001::a]:443"},
		{2, "[2001:67c:04e8:f002::a]:443"},
	}

	for _, tc := range cases {
		t.Run(tc.addr, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", tc.addr)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			t.Cleanup(func() { _ = conn.Close() })

			rtt, err := dcprobe.Probe(ctx, conn, tc.dc)
			if err != nil {
				t.Fatalf("probe DC %d: %v", tc.dc, err)
			}
			t.Logf("DC %d (%s): rtt=%s", tc.dc, tc.addr, rtt)
		})
	}
}

// TestProbeRejectsMisbehavingPeer connects to an in-process listener that
// accepts the obfs2 handshake, then writes back arbitrary bytes. With
// overwhelming probability the decrypted reply fails one of: frame-length
// bounds, resPQ constructor, or nonce echo. All three paths wrap
// ErrNotTelegram, so we assert errors.Is.
func TestProbeRejectsMisbehavingPeer(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close() //nolint: errcheck

		// Discard the 64-byte obfs2 handshake the client sends.
		var hs [64]byte
		if _, err := io.ReadFull(c, hs[:]); err != nil {
			return
		}
		// Write enough garbage to satisfy any plausible respLen the client
		// might decode (we cap at maxResPQFrame=256 in probe.go). Whatever
		// the client decrypts will fail constructor or nonce verification.
		junk := make([]byte, 512)
		for i := range junk {
			junk[i] = byte(i)
		}
		_, _ = c.Write(junk)
		// Keep the conn open until the client closes it (avoids racing the
		// client's read against our close).
		_, _ = io.Copy(io.Discard, c)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	_, err = dcprobe.Probe(ctx, conn, 2)
	if err == nil {
		t.Fatal("expected ErrNotTelegram, got nil")
	}
	if !errors.Is(err, dcprobe.ErrNotTelegram) {
		t.Fatalf("expected errors.Is(err, ErrNotTelegram) to be true, got: %v", err)
	}
	t.Logf("rejected: %v", err)
}
