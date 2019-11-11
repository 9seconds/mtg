package stats

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"mtg/config"
)

var Stats Interface

func Init(ctx context.Context) error {
	mux := http.NewServeMux()

	instancePrometheus, err := newStatsPrometheus(mux)
	if err != nil {
		return fmt.Errorf("cannot initialize prometheus: %w", err)
	}

	stats := []Interface{instancePrometheus}

	if config.C.StatsdAddr != nil {
		instanceStatsd, err := newStatsStatsd()
		if err != nil {
			return fmt.Errorf("cannot inialize statsd: %w", err)
		}

		stats = append(stats, instanceStatsd)
	}

	listener, err := net.Listen("tcp", config.C.StatsBind.String())
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

	Stats = multiStats(stats)

	return nil
}
