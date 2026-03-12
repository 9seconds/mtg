package doppel

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
	"github.com/9seconds/mtg/v2/mtglib/internal/tls"
)

const (
	// Please see Stats description
	// https://blog.cloudflare.com/optimizing-tls-over-tcp-to-reduce-latency/
	// https://github.com/cloudflare/sslconfig/blob/master/patches/nginx__dynamic_tls_records.patch
	TLSRecordSizeStart = 1369
	TLSRecordSizeAccel = 4229
	TLSRecordSizeMax   = 16384 - tls.SizeHeader

	TLSCounterAccelAfter = 40
	TLSCounterMaxAfter   = TLSCounterAccelAfter + 20

	TLSRecordSizeResetAfter = time.Second
)

// copypasted from mtglib
type Network interface {
	// Dial establishes context-free TCP connections.
	Dial(network, address string) (essentials.Conn, error)

	// DialContext dials using a context. This is a preferable way of
	// establishing TCP connections.
	DialContext(ctx context.Context, network, address string) (essentials.Conn, error)

	// MakeHTTPClient build an HTTP client with given dial function. If nothing is
	// provided, then DialContext of this interface is going to be used.
	MakeHTTPClient(func(ctx context.Context, network, address string) (essentials.Conn, error)) *http.Client

	// NativeDialer returns a configured instance of native dialer that
	// skips proxy connections or any other irrelevant settings.
	NativeDialer() *net.Dialer
}
