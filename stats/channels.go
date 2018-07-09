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
	CrashesChan     = make(chan struct{}, crashesChanLength)
	ConnectionsChan = make(chan *connectionData, connectionsChanLength)
	TrafficChan     = make(chan *trafficData, trafficChanLength)
)

type connectionData struct {
	connectionType mtproto.ConnectionType
	addr           *net.TCPAddr
	connected      bool
}

type trafficData struct {
	traffic int
	ingress bool
}

func crashManager() {
	for range CrashesChan {
		instance.mutex.RLock()

		instance.Crashes++

		instance.mutex.RUnlock()
	}
}

func connectionManager() {
	for event := range ConnectionsChan {
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
		case event := <-TrafficChan:
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

func NewCrash() {
	CrashesChan <- struct{}{}
}

func ClientConnected(connectionType mtproto.ConnectionType, addr *net.TCPAddr) {
	ConnectionsChan <- &connectionData{
		connectionType: connectionType,
		addr:           addr,
		connected:      true,
	}
}

func ClientDisconnected(connectionType mtproto.ConnectionType, addr *net.TCPAddr) {
	ConnectionsChan <- &connectionData{
		connectionType: connectionType,
		addr:           addr,
		connected:      false,
	}
}

func IngressTraffic(traffic int) {
	TrafficChan <- &trafficData{
		traffic: traffic,
		ingress: true,
	}
}

func EgressTraffic(traffic int) {
	TrafficChan <- &trafficData{
		traffic: traffic,
		ingress: false,
	}
}
