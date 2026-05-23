package cmd

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-go-golems/scraper/pkg/metrics"
	"github.com/stretchr/testify/require"
)

func TestMaybeStartWorkerMetricsServerServesMetrics(t *testing.T) {
	registry, err := metrics.NewRegistry()
	require.NoError(t, err)
	registry.SetWorkerUp("worker-test", true)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server, addr, err := maybeStartWorkerMetricsServer(ctx, "127.0.0.1:0", "/metrics", registry)
	require.NoError(t, err)
	require.NotNil(t, server)
	require.NotEmpty(t, addr)

	t.Cleanup(func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer shutdownCancel()
		_ = server.Shutdown(shutdownCtx)
	})

	require.Eventually(t, func() bool {
		resp, err := http.Get("http://" + addr + "/metrics")
		if err != nil {
			return false
		}
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusOK {
			return false
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}
		return strings.Contains(string(body), "scraper_workers_up")
	}, 2*time.Second, 25*time.Millisecond)
}
