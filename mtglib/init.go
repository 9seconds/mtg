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

	errCannotSendWelcomePacket = errors.New("cannot send welcome packet")
	errReplayAttackDetected    = errors.New("replay attack detected")
)

const (
	DefaultConcurrency = 4096
	DefaultBufferSize  = 16 * 1024 // 16 kib
	DefaultCloakPort   = 443
	DefaultIdleTimeout = time.Minute
	DefaultPreferIP    = "prefer-ipv6"
)

type Network interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	MakeHTTPClient(func(ctx context.Context, network, address string) (net.Conn, error)) *http.Client
}

type AntiReplayCache interface {
	SeenBefore(data []byte) bool
	Shutdown()
}

type IPBlocklist interface {
	Contains(net.IP) bool
	Shutdown()
}

type Event interface {
	StreamID() string
}

type EventStream interface {
	Send(context.Context, Event)
	Shutdown()
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
