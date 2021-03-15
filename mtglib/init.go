package mtglib

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"
)

var ErrSecretEmpty = errors.New("secret is empty")

type Network interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
	MakeHTTPClient(func(ctx context.Context, network, address string) (net.Conn, error)) *http.Client
	IdleTimeout() time.Duration
}

type AntiReplayCache interface {
	SeenBefore(data []byte) bool
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
