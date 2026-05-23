package cmd

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/rs/zerolog/log"
)

func maybeStartWorkerMetricsServer(ctx context.Context, addr string, path string, metricsRegistry *metrics.Registry) (*http.Server, string, error) {
	if addr == "" || metricsRegistry == nil {
		return nil, "", nil
	}
	if path == "" {
		path = "/metrics"
	}
	mux := http.NewServeMux()
	mux.Handle(path, metricsRegistry.Handler())
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, "", err
	}
	actualAddr := listener.Addr().String()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Warn().Err(err).Str("component", "worker-metrics").Str("address", actualAddr).Msg("worker metrics server stopped")
		}
	}()
	return server, actualAddr, nil
}
