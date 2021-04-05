package mtglib

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

var (
	ErrSecretEmpty                    = errors.New("secret is empty")
	ErrSecretInvalid                  = errors.New("secret is invalid")
	ErrNetworkIsNotDefined            = errors.New("network is not defined")
	ErrAntiReplayCacheIsNotDefined    = errors.New("anti-replay cache is not defined")
	ErrTimeAttackDetectorIsNotDefined = errors.New("time attack detector is not defined")
	ErrIPBlocklistIsNotDefined        = errors.New("ip blocklist is not defined")
	ErrEventStreamIsNotDefined        = errors.New("event stream is not defined")
	ErrLoggerIsNotDefined             = errors.New("logger is not defined")
)

const (
	DefaultConcurrency        = 4096
	DefaultBufferSize         = 16 * 1024 // 16 kib
	DefaultDomainFrontingPort = 443
	DefaultIdleTimeout        = time.Minute
	DefaultPreferIP           = "prefer-ipv6"
)

type Network interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	MakeHTTPClient(func(ctx context.Context, network, address string) (net.Conn, error)) *http.Client
}

// AntiReplayCache is an interface that is used to detect replay attacks
// based on some traffic fingerprints.
//
// Replay attacks are probe attacks whose main goal is to identify if
// server software can be classified in some way. For example, if you
// send some HTTP request to a web server, then you can expect that this
// server will respond with HTTP response back.
//
// There is a problem though. Let's imagine, that connection is
// encrypted. Let's imagine, that it is encrypted with some static key
// like ShadowSocks (https://shadowsocks.org/assets/whitepaper.pdf).
// In that case, in theory, if you repeat the same bytes, you can get
// the same responses. Let's imagine, that you've cracked the key. then
// if you send the same bytes, you can decrypt a response and see its
// structure. Based on its structure you can identify if this server is
// SOCKS5, MTPROTO proxy etc.
//
// This is just one example, maybe not the best or not the most
// relevant. In real life, different organizations use such replay
// attacks to perform some reverse engineering of the proxy, do some
// statical analysis to identify server software.
//
// There are many ways how to protect your proxy against them. One
// is domain fronting which is a core part of mtg. Another one is to
// collect some 'handshake fingerprints' and forbid duplication.
//
// So, it one is sending the same byte flow right after you (or a couple
// of hours after), mtg should detect that and reject this connection
// (or redirect to fronting domain).
type AntiReplayCache interface {
	// Seen before checks if this set of bytes was observed before or
	// not. If it is required to store this information somewhere else,
	// then it has to do that.
	SeenBefore(data []byte) bool
}

type IPBlocklist interface {
	Contains(net.IP) bool
}

type Event interface {
	StreamID() string
	Timestamp() time.Time
}

type EventStream interface {
	Send(context.Context, Event)
}

type TimeAttackDetector interface {
	Valid(time.Time) error
}

type Logger interface {
	Named(name string) Logger

	BindInt(name string, value int) Logger
	BindStr(name, value string) Logger

	Printf(format string, args ...interface{})
	Info(msg string)
	InfoError(msg string, err error)
	Warning(msg string)
	WarningError(msg string, err error)
	Debug(msg string)
	DebugError(msg string, err error)
}
