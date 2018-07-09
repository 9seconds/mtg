package stats

import (
	"net"
	"time"

	"github.com/9seconds/mtg/mtproto"
)

const (
	crashesChanLength     = 1
	connectionsChanLength = 20
	trafficChanLength     = 5000
)

var (
	crashesChan     = make(chan struct{}, crashesChanLength)
	connectionsChan = make(chan *connectionData, connectionsChanLength)
	trafficChan     = make(chan *trafficData, trafficChanLength)
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

func crashManager() {
	for range crashesChan {
		instance.mutex.RLock()

		instance.Crashes++

		instance.mutex.RUnlock()
	}
}

func connectionManager() { // nolint: gocyclo
	for event := range connectionsChan {
		instance.mutex.RLock()

		isIPv4 := event.addr.IP.To4() != nil
		var inc uint32 = 1
		if !event.connected {
			inc = ^uint32(0)
		}

		switch event.connectionType {
		case mtproto.ConnectionTypeAbridged:
			if isIPv4 {
				instance.ActiveConnections.Abridged.IPv4 += inc
				if event.connected {
					instance.AllConnections.Abridged.IPv4 += inc
				}
			} else {
				instance.ActiveConnections.Abridged.IPv6 += inc
				if event.connected {
					instance.AllConnections.Abridged.IPv6 += inc
				}
			}
		default:
			if isIPv4 {
				instance.ActiveConnections.Intermediate.IPv4 += inc
				if event.connected {
					instance.AllConnections.Intermediate.IPv4 += inc
				}
			} else {
				instance.ActiveConnections.Intermediate.IPv6 += inc
				if event.connected {
					instance.AllConnections.Intermediate.IPv6 += inc
				}
			}
		}

		instance.mutex.RUnlock()
	}
}

func trafficManager() {
	speedChan := time.Tick(time.Second)

	for {
		select {
		case event := <-trafficChan:
			instance.mutex.RLock()

			if event.ingress {
				instance.Traffic.Ingress += trafficValue(event.traffic)
				instance.speedCurrent.Ingress += trafficSpeedValue(event.traffic)
			} else {
				instance.Traffic.Egress += trafficValue(event.traffic)
				instance.speedCurrent.Egress += trafficSpeedValue(event.traffic)
			}

			instance.mutex.RUnlock()
		case <-speedChan:
			instance.mutex.RLock()

			instance.Speed.Ingress = instance.speedCurrent.Ingress
			instance.Speed.Egress = instance.speedCurrent.Egress
			instance.speedCurrent.Ingress = trafficSpeedValue(0)
			instance.speedCurrent.Egress = trafficSpeedValue(0)

			instance.mutex.RUnlock()
		}
	}
}

// NewCrash indicates new crash.
func NewCrash() {
	crashesChan <- struct{}{}
}

// ClientConnected indicates that new client was connected.
func ClientConnected(connectionType mtproto.ConnectionType, addr *net.TCPAddr) {
	connectionsChan <- &connectionData{
		connectionType: connectionType,
		addr:           addr,
		connected:      true,
	}
}

// ClientDisconnected indicates that client was disconnected.
func ClientDisconnected(connectionType mtproto.ConnectionType, addr *net.TCPAddr) {
	connectionsChan <- &connectionData{
		connectionType: connectionType,
		addr:           addr,
		connected:      false,
	}
}

// IngressTraffic accounts new ingress traffic.
func IngressTraffic(traffic int) {
	trafficChan <- &trafficData{
		traffic: traffic,
		ingress: true,
	}
}

// EgressTraffic accounts new ingress traffic.
func EgressTraffic(traffic int) {
	trafficChan <- &trafficData{
		traffic: traffic,
		ingress: false,
	}
}
