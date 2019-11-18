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

	stats := []Interface{newStatsPrometheus(mux)}
	if config.C.StatsdAddr != nil {
		stats = append(stats, newStatsStatsd())
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
