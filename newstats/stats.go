package newstats

import (
	"net"
	"net/http"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/newconfig"
	"github.com/9seconds/mtg/newprotocol"
)

type Stats interface {
	IngressTraffic(int)
	EgressTraffic(int)
	ClientConnected(newprotocol.ConnectionType, *net.TCPAddr)
	ClientDisconnected(newprotocol.ConnectionType, *net.TCPAddr)
	Crash()
	AntiReplayDetected()
}

type multiStats []Stats

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

func (m multiStats) ClientConnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	for i := range m {
		go m[i].ClientConnected(connectionType, addr)
	}
}

func (m multiStats) ClientDisconnected(connectionType newprotocol.ConnectionType, addr *net.TCPAddr) {
	for i := range m {
		go m[i].ClientDisconnected(connectionType, addr)
	}
}

func (m multiStats) Crash() {
	for i := range m {
		go m[i].Crash()
	}
}

func (m multiStats) AntiReplayDetected() {
	for i := range m {
		go m[i].AntiReplayDetected()
	}
}

var S Stats

func Init() error {
	mux := http.NewServeMux()

	instanceJSON := newStatsJSON(mux)
	instancePrometheus, err := newStatsPrometheus(mux)
	if err != nil {
		return errors.Annotate(err, "Cannot initialize Prometheus")
	}

	stats := []Stats{instanceJSON, instancePrometheus}
	if newconfig.C.StatsdStats.Addr.IP != nil {
		instanceStatsd, err := newStatsStatsd()
		if err != nil {
			return errors.Annotate(err, "Cannot initialize StatsD")
		}
		stats = append(stats, instanceStatsd)
	}

	listener, err := net.Listen("tcp", newconfig.C.StatsAddr.String())
	if err != nil {
		return errors.Annotate(err, "Cannot initialize stats server")
	}

	srv := http.Server{
		Handler: mux,
	}
	go srv.Serve(listener) // nolint: errcheck

	S = multiStats(stats)

	return nil
}
