package stats

import (
	"net"

	"github.com/9seconds/mtg/conntypes"
)

type IngressTrafficInterface interface {
	IngressTraffic(int)
}

type EgressTrafficInterface interface {
	EgressTraffic(int)
}

type ClientConnectedInterface interface {
	ClientConnected(conntypes.ConnectionType, *net.TCPAddr)
}

type ClientDisconnectedInterface interface {
	ClientDisconnected(conntypes.ConnectionType, *net.TCPAddr)
}

type TelegramConnectedInterface interface {
	TelegramConnected(conntypes.DC, *net.TCPAddr)
}

type TelegramDisconnectedInterface interface {
	TelegramDisconnected(conntypes.DC, *net.TCPAddr)
}

type CrashInterface interface {
	Crash()
}

type AntiReplayDetectedInterface interface {
	AntiReplayDetected()
}

type Interface interface {
	IngressTrafficInterface
	EgressTrafficInterface
	ClientConnectedInterface
	ClientDisconnectedInterface
	TelegramConnectedInterface
	TelegramDisconnectedInterface
	CrashInterface
	AntiReplayDetectedInterface
}
