package cli

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/internal/config"
	"github.com/9seconds/mtg/v2/mtglib"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/dns/dnsmessage"
)

// startSNITestDNS spins up a loopback UDP resolver that answers every query
// with the given A and AAAA records, so runSNICheck sees a dual-stack secret
// host without touching the real network. It returns a *net.Resolver wired to
// it.
func startSNITestDNS(t *testing.T, a, aaaa net.IP) *net.Resolver {
	t.Helper()

	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { pc.Close() }) //nolint: errcheck

	go func() {
		buf := make([]byte, 512)

		for {
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}

			var parser dnsmessage.Parser

			hdr, err := parser.Start(buf[:n])
			if err != nil {
				continue
			}

			question, err := parser.Question()
			if err != nil {
				continue
			}

			builder := dnsmessage.NewBuilder(nil, dnsmessage.Header{
				ID:                 hdr.ID,
				Response:           true,
				RecursionAvailable: true,
			})
			builder.EnableCompression()
			_ = builder.StartQuestions()
			_ = builder.Question(question)
			_ = builder.StartAnswers()

			rh := dnsmessage.ResourceHeader{
				Name:  question.Name,
				Class: dnsmessage.ClassINET,
				TTL:   60,
			}

			switch question.Type {
			case dnsmessage.TypeA:
				rh.Type = dnsmessage.TypeA
				var v4 [4]byte
				copy(v4[:], a.To4())
				_ = builder.AResource(rh, dnsmessage.AResource{A: v4})
			case dnsmessage.TypeAAAA:
				rh.Type = dnsmessage.TypeAAAA
				var v6 [16]byte
				copy(v6[:], aaaa.To16())
				_ = builder.AAAAResource(rh, dnsmessage.AAAAResource{AAAA: v6})
			}

			msg, err := builder.Finish()
			if err != nil {
				continue
			}

			pc.WriteTo(msg, addr) //nolint: errcheck
		}
	}()

	dnsAddr := pc.LocalAddr().String()

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, _, _ string) (net.Conn, error) {
			var d net.Dialer

			return d.DialContext(ctx, "udp", dnsAddr)
		},
	}
}

// ipv4OnlyEgressNetwork fakes mtglib.Network so that public-IP detection
// succeeds over tcp4 and fails over tcp6 — the classic IPv4-only-egress
// server. getIP's per-protocol dial is routed at a loopback listener: a tcp4
// dial to 127.0.0.1 connects, a tcp6 dial to the same address fails ("no
// suitable address"), so we exercise the real getIP code path without the
// internet.
type ipv4OnlyEgressNetwork struct {
	listenerAddr string
	detectedV4   string
}

func (n *ipv4OnlyEgressNetwork) Dial(_, _ string) (essentials.Conn, error) {
	panic("unused")
}

func (n *ipv4OnlyEgressNetwork) DialContext(_ context.Context, _, _ string) (essentials.Conn, error) {
	panic("unused")
}

func (n *ipv4OnlyEgressNetwork) NativeDialer() *net.Dialer {
	return &net.Dialer{}
}

func (n *ipv4OnlyEgressNetwork) MakeHTTPClient(
	dialFunc func(ctx context.Context, network, address string) (essentials.Conn, error),
) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			conn, err := dialFunc(req.Context(), "tcp", n.listenerAddr)
			if err != nil {
				return nil, err
			}

			conn.Close() //nolint: errcheck

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(n.detectedV4)),
				Header:     make(http.Header),
			}, nil
		}),
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// TestRunSNICheckIPv4OnlyEgressGraceful reproduces the #529/#542 regression:
// a dual-stack secret host on a server whose IPv6 egress is down. The tcp6
// public-IP detection fails, but the tcp4 detection succeeds and matches the
// host's A record, so the SNI check must NOT report a hard error — one
// family being undetectable is graceful degradation, not failure.
func TestRunSNICheckIPv4OnlyEgressGraceful(t *testing.T) {
	const ourV4 = "192.0.2.4" // RFC 5737 TEST-NET-1

	resolver := startSNITestDNS(t, net.ParseIP(ourV4), net.ParseIP("2001:db8::1")) // RFC 3849 doc range

	// Loopback target for getIP's dial. Keep it the IPv4 literal 127.0.0.1: a
	// "tcp6" dial to it fails deterministically ("no suitable address") on any
	// host regardless of IPv6 connectivity, which is what makes tcp6 detection
	// fail here. Do not replace with a real ::1 setup — that reintroduces flake.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { listener.Close() }) //nolint: errcheck

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			conn.Close() //nolint: errcheck
		}
	}()

	ntw := &ipv4OnlyEgressNetwork{
		listenerAddr: listener.Addr().String(),
		detectedV4:   ourV4,
	}

	conf := &config.Config{}
	conf.Secret.Host = "secret-host.test"

	res, err := runSNICheck(context.Background(), conf, resolver, ntw)

	// The load-bearing assertion: a single family's detection failure must not
	// poison the whole result. Before the fix this returns a non-nil error.
	require.NoError(t, err)
	require.Equal(t, ourV4, res.OurIP4, "IPv4 public IP should be detected and match the A record")
	require.Empty(t, res.OurIP6, "IPv6 is undetectable on IPv4-only egress; must degrade, not error")
}

var _ mtglib.Network = (*ipv4OnlyEgressNetwork)(nil)
