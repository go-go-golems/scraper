package server

import (
	"net/http"
	"strings"

	"github.com/go-go-golems/scraper/pkg/api/handlers"
	"github.com/go-go-golems/scraper/pkg/metrics"
)

func registerEngineRoutes(mux *http.ServeMux, engineHandler *handlers.EngineHandler, metricsRegistry *metrics.Registry) {
	mux.Handle("GET /metrics", metricsRegistry.Handler())
	mux.HandleFunc("GET /api/v1/engine/status", engineHandler.Status)
	mux.HandleFunc("GET /api/v1/engine/migrations", engineHandler.Migrations)
	mux.HandleFunc("GET /api/v1/workflows", engineHandler.Workflows)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}", engineHandler.Workflow)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/ops", engineHandler.WorkflowOps)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/artifacts", engineHandler.WorkflowArtifacts)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/ops/{opID}/result", engineHandler.OpResult)
	mux.HandleFunc("GET /api/v1/workflows/{workflowID}/ops/{opID}/artifacts", engineHandler.OpArtifacts)
	mux.HandleFunc("GET /api/v1/artifacts/{artifactID}", engineHandler.ArtifactDownload)
	mux.HandleFunc("GET /api/v1/queues", engineHandler.Queues)
	mux.HandleFunc("POST /api/v1/workflows/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/workflows/")
		if strings.HasSuffix(path, ":cancel") {
			wfID := strings.TrimSuffix(path, ":cancel")
			r.SetPathValue("workflowID", wfID)
			engineHandler.CancelWorkflow(w, r)
			return
		}
		parts := strings.Split(path, "/")
		if len(parts) == 3 && parts[1] == "ops" && strings.HasSuffix(parts[2], ":retry") {
			r.SetPathValue("workflowID", parts[0])
			r.SetPathValue("opID", strings.TrimSuffix(parts[2], ":retry"))
			engineHandler.RetryOp(w, r)
			return
		}
		http.NotFound(w, r)
	})
}
