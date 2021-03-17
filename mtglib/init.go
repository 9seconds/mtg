package mtglib

import (
	"context"
	"errors"
	"net"
	"net/http"
)

var (
	ErrSecretEmpty                 = errors.New("secret is empty")
	ErrSecretInvalid               = errors.New("secret is invalid")
	ErrNetworkIsNotDefined         = errors.New("network is not defined")
	ErrAntiReplayCacheIsNotDefined = errors.New("anti-replay cache is not defined")
	ErrIPBlocklistIsNotDefined     = errors.New("ip blocklist is not defined")
	ErrEventStreamIsNotDefined     = errors.New("event stream is not defined")
	ErrLoggerIsNotDefined          = errors.New("logger is not defined")
)

const (
	DefaultConcurrency = 4096
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

type Logger interface {
	Named(name string) Logger

	BindInt(name string, value int) Logger
	BindStr(name, value string) Logger

	Info(msg string)
	InfoError(msg string, err error)
	Warning(msg string)
	WarningError(msg string, err error)
	Debug(msg string)
	DebugError(msg string, err error)
}
