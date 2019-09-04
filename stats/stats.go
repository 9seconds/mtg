package stats

import (
	"context"
	"net"
	"net/http"

	"github.com/juju/errors"

	"github.com/9seconds/mtg/config"
	"github.com/9seconds/mtg/conntypes"
)

type Stats interface {
	IngressTraffic(int)
	EgressTraffic(int)
	ClientConnected(conntypes.ConnectionType, *net.TCPAddr)
	ClientDisconnected(conntypes.ConnectionType, *net.TCPAddr)
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

func Init(ctx context.Context) error {
	mux := http.NewServeMux()

	instanceJSON := newStatsJSON(mux)
	instancePrometheus, err := newStatsPrometheus(mux)
	if err != nil {
		return errors.Annotate(err, "Cannot initialize Prometheus")
	}

	stats := []Stats{instanceJSON, instancePrometheus}
	if config.C.StatsdStats.Addr.IP != nil {
		instanceStatsd, err := newStatsStatsd()
		if err != nil {
			return errors.Annotate(err, "Cannot initialize StatsD")
		}
		stats = append(stats, instanceStatsd)
	}

	listener, err := net.Listen("tcp", config.C.StatsAddr.String())
	if err != nil {
		return errors.Annotate(err, "Cannot initialize stats server")
	}

	srv := http.Server{
		Handler: mux,
	}
	go srv.Serve(listener) // nolint: errcheck
	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background()) // nolint: errcheck
	}()

	S = multiStats(stats)

	return nil
}
