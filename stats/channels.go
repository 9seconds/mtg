package stats

import (
	"net"

	"github.com/9seconds/mtg/mtproto"
)

const (
	connectionsChanLength = 10
	trafficChanLength     = 10
)

var (
	crashesChan     = make(chan struct{})
	statsChan       = make(chan chan<- Stats)
	connectionsChan = make(chan connectionData, connectionsChanLength)
	trafficChan     = make(chan trafficData, trafficChanLength)
)

type connectionData struct {
	connectionType mtproto.ConnectionType
	connected      bool
	addr           *net.TCPAddr
}

type trafficData struct {
	traffic int
	ingress bool
}

// NewCrash indicates new crash.
func NewCrash() {
	crashesChan <- struct{}{}
}

// ClientConnected indicates that new client was connected.
func ClientConnected(connectionType mtproto.ConnectionType, addr *net.TCPAddr) {
	connectionsChan <- connectionData{
		connectionType: connectionType,
		addr:           addr,
		connected:      true,
	}
}

// ClientDisconnected indicates that client was disconnected.
func ClientDisconnected(connectionType mtproto.ConnectionType, addr *net.TCPAddr) {
	connectionsChan <- connectionData{
		connectionType: connectionType,
		addr:           addr,
		connected:      false,
	}
}

// IngressTraffic accounts new ingress traffic.
func IngressTraffic(traffic int) {
	trafficChan <- trafficData{
		traffic: traffic,
		ingress: true,
	}
}

// EgressTraffic accounts new ingress traffic.
func EgressTraffic(traffic int) {
	trafficChan <- trafficData{
		traffic: traffic,
		ingress: false,
	}
}

func GetStats() Stats {
	rpcChan := make(chan Stats)
	statsChan <- rpcChan
	return <-rpcChan
}
