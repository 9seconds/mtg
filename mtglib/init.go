// mtglib defines a package with MTPROTO proxy.
//
// Since mtg itself is build as an example of how to work with mtglib, it worth
// to telling a couple of words about a project organization.
//
// A core object of the project is [mtglib.Proxy]. This is a proxy you expect:
// that one which you configure, set to serve on a listener and/or shutdown on
// application termination.
//
// But it also has a core logic unrelated to Telegram per se: anti replay
// cache, network connectivity (who knows, maybe you want to have a native
// VMESS integration) and so on.
//
// You can supply such parts to a proxy with interfaces. The rest of the
// packages in mtg define some default implementations of these interfaces. But
// if you want to integrate it with, let say, influxdb, you can do it easily.
package mtglib

import (
	"context"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/9seconds/mtg/v2/essentials"
)

var (
	// ErrSecretEmpty is returned if you are trying to create a proxy but do not
	// provide a secret.
	ErrSecretEmpty = errors.New("secret is empty")

	// ErrSecretInvalid is returned if you are trying to create a proxy but secret
	// value is invalid (no host or payload are zeroes).
	ErrSecretInvalid = errors.New("secret is invalid")

	// ErrNetworkIsNotDefined is returned if you are trying to create a proxy but
	// network value is undefined.
	ErrNetworkIsNotDefined = errors.New("network is not defined")

	// ErrAntiReplayCacheIsNotDefined is returned if you are trying to create a
	// proxy but anti replay cache value is undefined.
	ErrAntiReplayCacheIsNotDefined = errors.New("anti-replay cache is not defined")

	// ErrIPBlocklistIsNotDefined is returned if you are trying to create a proxy
	// but ip blocklist instance is not defined.
	ErrIPBlocklistIsNotDefined = errors.New("ip blocklist is not defined")

	// ErrIPAllowlistIsNotDefined is returned if you are trying to create a proxy
	// but ip allowlist instance is not defined.
	ErrIPAllowlistIsNotDefined = errors.New("ip allowlist is not defined")

	// ErrEventStreamIsNotDefined is returned if you are trying to create a proxy
	// but event stream instance is not defined.
	ErrEventStreamIsNotDefined = errors.New("event stream is not defined")

	// ErrLoggerIsNotDefined is returned if you are trying to create a proxy but
	// logger is not defined.
	ErrLoggerIsNotDefined = errors.New("logger is not defined")
)

const (
	// DefaultConcurrency is a default max count of simultaneously connected
	// clients.
	DefaultConcurrency = 4096

	// DefaultBufferSize is a default size of a copy buffer.
	//
	// Deprecated: this setting no longer makes any effect.
	DefaultBufferSize = 16 * 1024 // 16 kib

	// DefaultDomainFrontingPort is a default port (HTTPS) to connect to in case
	// of probe-resistance activity.
	DefaultDomainFrontingPort = 443

	// DefaultIdleTimeout is a default timeout for closing a connection in case of
	// idling.
	//
	// Deprecated: no longer in use because of changed TCP relay algorithm.
	DefaultIdleTimeout = time.Minute

	// DefaultTolerateTimeSkewness is a default timeout for time skewness on a
	// faketls timeout verification.
	DefaultTolerateTimeSkewness = 3 * time.Second

	// DefaultPreferIP is a default value for Telegram IP connectivity preference.
	DefaultPreferIP = "prefer-ipv6"

	// SecretKeyLength defines a length of the secret bytes used by Telegram and a
	// proxy.
	SecretKeyLength = 16

	// ConnectionIDBytesLength defines a count of random bytes used to generate a
	// stream/connection ids.
	ConnectionIDBytesLength = 16

	// TCPRelayReadTimeout defines a max time period between two consecuitive
	// reads from Telegram after which connection will be terminated. This is
	// required to abort stale connections.
	TCPRelayReadTimeout = 20 * time.Second
)

// Network defines a knowledge how to work with a network. It may sound fun but
// it encapsulates all the knowledge how to properly establish connections to
// remote hosts and configure HTTP clients.
//
// For example, if you want to use SOCKS5 proxy, you probably want to have all
// traffic routed to this proxy: telegram connections, http requests and so on.
// This knowledge is encapsulated into instances of such interface.
//
// mtglib uses Network for:
//  1. Dialing to Telegram
//  2. Dialing to front domain
//  3. Doing HTTP requests (for example, for FireHOL ipblocklist).
type Network interface {
	// Dial establishes context-free TCP connections.
	Dial(network, address string) (essentials.Conn, error)

	// DialContext dials using a context. This is a preferrable way of
	// establishing TCP connections.
	DialContext(ctx context.Context, network, address string) (essentials.Conn, error)

	// MakeHTTPClient build an HTTP client with given dial function. If nothing is
	// provided, then DialContext of this interface is going to be used.
	MakeHTTPClient(func(ctx context.Context, network, address string) (essentials.Conn, error)) *http.Client
}

// AntiReplayCache is an interface that is used to detect replay attacks based
// on some traffic fingerprints.
//
// Replay attacks are probe attacks whose main goal is to identify if server
// software can be classified in some way. For example, if you send some HTTP
// request to a web server, then you can expect that this server will respond
// with HTTP response back.
//
// There is a problem though. Let's imagine, that connection is encrypted.
// Let's imagine, that it is encrypted with some static key like [ShadowSocks].
// In that case, in theory, if you repeat the same bytes, you can get the same
// responses. Let's imagine, that you've cracked the key. then if you send the
// same bytes, you can decrypt a response and see its structure. Based on its
// structure you can identify if this server is SOCKS5, MTPROTO proxy etc.
//
// This is just one example, maybe not the best or not the most relevant. In
// real life, different organizations use such replay attacks to perform some
// reverse engineering of the proxy, do some statical analysis to identify
// server software.
//
// There are many ways how to protect your proxy against them. One is domain
// fronting which is a core part of mtg. Another one is to collect some
// 'handshake fingerprints' and forbid duplication.
//
// So, it one is sending the same byte flow right after you (or a couple of
// hours after), mtg should detect that and reject this connection (or redirect
// to fronting domain).
//
// [ShadowSocks]: https://shadowsocks.org/assets/whitepaper.pdf
type AntiReplayCache interface {
	// Seen before checks if this set of bytes was observed before or not. If it
	// is required to store this information somewhere else, then it has to do
	// that.
	SeenBefore(data []byte) bool
}

// IPBlocklist filters requests based on IP address.
//
// If this filter has an IP address, then mtg closes a request without reading
// anything from a socket. It also does not give such request to a worker pool,
// so in worst cases you can expect that you invoke this object more frequent
// than defined proxy concurrency.
type IPBlocklist interface {
	// Contains checks if given IP address belongs to this blocklist If. it is, a
	// connection is terminated .
	Contains(net.IP) bool

	// Run starts a background update procedure for a blocklist
	Run(time.Duration)

	// Shutdown stops a blocklist. It is assumed that none will access it after.
	Shutdown()
}

// Event is a data structure which is populated during mtg request processing
// lifecycle. Each request popluates many events:
//  1. Client connected
//  2. Request is finished
//  3. Connection to Telegram server is established
//
// and so on. All these events are data structures but all of them must conform
// the same interface.
type Event interface {
	// StreamID returns an identifier of the stream, connection, request, you name
	// it. All events within the same stream returns the same stream id.
	StreamID() string

	// Timestamp returns a timestamp when this event was generated.
	Timestamp() time.Time
}

// EventStream is an abstraction that accepts a set of events produced by mtg.
// Its main goal is to inject your logging or monitoring system.
//
// The idea is simple. When mtg works, it emits a set of events during a
// lifecycle of the requestor: EventStart, EventFinish etc. mtg is a producer
// which puts these events into a stream. Responsibility of the stream is to
// deliver this event to consumers/observers. There might be many different
// observers (for example, you want to have both statsd and prometheus), mtg
// should know nothing about them.
type EventStream interface {
	// Send delivers an event to observers. Given context has to be respected. If
	// the context is closed, all blocking operations should be released ASAP.
	//
	// It is possible that context is closed but the message is delivered.
	// EventStream implementations should solve this issue somehow.
	Send(context.Context, Event)
}

// Logger defines an interface of the logger used by mtglib.
//
// Each logger has a name. It is possible to stack names to organize poor-man
// namespaces. Also, each logger must be able to bind parameters to avoid
// pushing them all the time.
//
// Example
//
//	logger := SomeLogger{} logger = logger.BindStr("ip", net.IP{127, 0, 0, 1})
//	logger.Info("Hello")
//
// In that case, ip is bound as a parameter. It is a great idea to put this
// parameter somewhere in a log message.
//
//	logger1 = logger.BindStr("param1", "11") logger2 = logger.BindInt("param2",
//	11)
//
// logger1 should see no param2 and vice versa, logger2 should not see param1
// If you attach a parameter to a logger, parents should not know about that.
type Logger interface {
	// Named returns a new logger with a bound name. Name chaining is allowed and
	// appreciated.
	Named(name string) Logger

	// BindInt binds new integer parameter to a new logger instance.
	BindInt(name string, value int) Logger

	// BindStr binds new string parameter to a new logger instance.
	BindStr(name, value string) Logger

	// BindJSON binds a new JSON-encoded string to a new logger instance.
	BindJSON(name, value string) Logger

	// Printf is to support log.Logger behavior.
	Printf(format string, args ...interface{})

	// Info puts a message about some normal situation.
	Info(msg string)

	// InfoError puts a message about some normal situation but this situation is
	// related to a given error.
	InfoError(msg string, err error)

	// Warning puts a message about some extraordinary situation worth to look at.
	Warning(msg string)

	// WarningError puts a message about some extraordinary situation worth to
	// look at. This situation is related to a given error.
	WarningError(msg string, err error)

	// Debug puts a message useful for debugging only.
	Debug(msg string)

	// Debug puts a message useful for debugging only. This message is related to
	// a given error.
	DebugError(msg string, err error)
}
