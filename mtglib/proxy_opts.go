package mtglib

type ProxyOpts struct {
	Secret          Secret
	Network         Network
	AntiReplayCache AntiReplayCache
	IPBlocklist     IPBlocklist
	EventStream     EventStream
	Logger          Logger

	Concurrency uint
}
