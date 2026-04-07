package server

import (
	"net/http"
	"strings"

	"github.com/go-go-golems/scraper/pkg/api/handlers"
)

func registerCatalogRoutes(mux *http.ServeMux, catalogHandler *handlers.CatalogHandler, submissionHandler *handlers.SubmissionHandler) {
	mux.HandleFunc("GET /healthz", catalogHandler.Healthz)
	mux.HandleFunc("GET /api/v1/info", catalogHandler.Info)
	mux.HandleFunc("GET /api/v1/sites", catalogHandler.Sites)
	mux.HandleFunc("GET /api/v1/sites/{site}", catalogHandler.Site)
	mux.HandleFunc("GET /api/v1/sites/{site}/detail", catalogHandler.SiteDetail)
	mux.HandleFunc("GET /api/v1/sites/{site}/verbs", catalogHandler.Verbs)
	mux.HandleFunc("GET /api/v1/sites/{site}/verbs/{verb}", catalogHandler.Verb)
	mux.HandleFunc("GET /api/v1/sites/{site}/scripts", catalogHandler.Scripts)
	mux.HandleFunc("GET /api/v1/sites/{site}/scripts/{path...}", catalogHandler.Script)
	mux.HandleFunc("POST /api/v1/sites/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, ":submit") {
			http.NotFound(w, r)
			return
		}
		trimmed := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/sites/"), ":submit")
		parts := strings.Split(trimmed, "/")
		if len(parts) != 3 || parts[1] != "verbs" {
			http.NotFound(w, r)
			return
		}
		r.SetPathValue("site", parts[0])
		r.SetPathValue("verb", parts[2])
		submissionHandler.Submit(w, r)
	})
}
