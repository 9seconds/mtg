package mtglib

import "time"

type ProxyOpts struct {
	Secret             Secret
	Network            Network
	AntiReplayCache    AntiReplayCache
	TimeAttackDetector TimeAttackDetector
	IPBlocklist        IPBlocklist
	EventStream        EventStream
	Logger             Logger

	BufferSize  uint
	Concurrency uint
	CloakPort   uint
	IdleTimeout time.Duration
	PreferIP    string
}
