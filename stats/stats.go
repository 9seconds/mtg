package stats

import (
	"context"
	"fmt"
	"net"
	"net/http"

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
		return fmt.Errorf("cannot initialize prometheus: %w", err)
	}

	stats := []Stats{instanceJSON, instancePrometheus}
	if config.C.StatsdStats.Addr.IP != nil {
		instanceStatsd, err := newStatsStatsd()
		if err != nil {
			return fmt.Errorf("cannot inialize statsd: %w", err)
		}
		stats = append(stats, instanceStatsd)
	}

	listener, err := net.Listen("tcp", config.C.StatsAddr.String())
	if err != nil {
		return fmt.Errorf("cannot initialize stats server: %w", err)
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
