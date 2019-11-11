package stats

import (
	"net"

	"github.com/9seconds/mtg/conntypes"
)

type multiStats []Interface

func (m multiStats) IngressTraffic(traffic int) {
	for i := range m {
		go m[i].IngressTraffic(traffic)
	}
}

func (m multiStats) EgressTraffic(traffic int) {
	for i := range m {
		go m[i].EgressTraffic(traffic)
	}
}

func (m multiStats) ClientConnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	for i := range m {
		go m[i].ClientConnected(connectionType, addr)
	}
}

func (m multiStats) ClientDisconnected(connectionType conntypes.ConnectionType, addr *net.TCPAddr) {
	for i := range m {
		go m[i].ClientDisconnected(connectionType, addr)
	}
}

func (m multiStats) TelegramConnected(dc conntypes.DC, addr *net.TCPAddr) {
	for i := range m {
		go m[i].TelegramConnected(dc, addr)
	}
}

func (m multiStats) TelegramDisconnected(dc conntypes.DC, addr *net.TCPAddr) {
	for i := range m {
		go m[i].TelegramDisconnected(dc, addr)
	}
}

func (m multiStats) Crash() {
	for i := range m {
		go m[i].Crash()
	}
}

func (m multiStats) ReplayDetected() {
	for i := range m {
		go m[i].ReplayDetected()
	}
}
