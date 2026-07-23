package workflowv3product

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type HTTPOptions struct {
	OperatorToken string
}

func NewHTTPHandler(app *Application, options HTTPOptions) (http.Handler, error) {
	if app == nil || app.Store == nil || app.Dispatcher == nil || app.Authoring == nil {
		return nil, errors.New("workflow application is required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v3/workflow/health", func(writer http.ResponseWriter, _ *http.Request) {
		writeHTTPJSON(writer, http.StatusOK, map[string]any{"ok": true})
	})
	mux.HandleFunc("GET /api/v3/workflow/task-packages", func(writer http.ResponseWriter, _ *http.Request) {
		writeHTTPJSON(writer, http.StatusOK, app.Authoring.Packages.Info())
	})
	mux.HandleFunc("GET /api/v3/workflow/runs", func(writer http.ResponseWriter, request *http.Request) {
		limit := 100
		if value := request.URL.Query().Get("limit"); value != "" {
			parsed, err := strconv.Atoi(value)
			if err != nil {
				writeHTTPError(writer, http.StatusBadRequest, "limit must be an integer")
				return
			}
			limit = parsed
		}
		runs, err := app.ListRuns(request.Context(), request.URL.Query().Get("status"), limit)
		if err != nil {
			writeHTTPError(writer, http.StatusBadRequest, err.Error())
			return
		}
		writeHTTPJSON(writer, http.StatusOK, runs)
	})
	mux.HandleFunc("GET /api/v3/workflow/runs/{runID}", func(writer http.ResponseWriter, request *http.Request) {
		runID := strings.TrimSpace(request.PathValue("runID"))
		if runID == "" {
			writeHTTPError(writer, http.StatusBadRequest, "run ID is required")
			return
		}
		view, err := app.Show(request.Context(), workflowv3.RunID(runID))
		if err != nil {
			writeHTTPError(writer, http.StatusNotFound, err.Error())
			return
		}
		writeHTTPJSON(writer, http.StatusOK, view)
	})
	mux.HandleFunc("POST /api/v3/workflow/runs/{runID}/cancel", func(writer http.ResponseWriter, request *http.Request) {
		if !validOperatorToken(request, options.OperatorToken) {
			writeHTTPError(writer, http.StatusForbidden, "operator authorization is required")
			return
		}
		runID := strings.TrimSpace(request.PathValue("runID"))
		if runID == "" {
			writeHTTPError(writer, http.StatusBadRequest, "run ID is required")
			return
		}
		view, err := app.Cancel(request.Context(), workflowv3.RunID(runID))
		if err != nil {
			writeHTTPError(writer, http.StatusConflict, err.Error())
			return
		}
		writeHTTPJSON(writer, http.StatusOK, view)
	})
	return mux, nil
}

func validOperatorToken(request *http.Request, expected string) bool {
	if expected == "" {
		return false
	}
	const prefix = "Bearer "
	provided := request.Header.Get("Authorization")
	if !strings.HasPrefix(provided, prefix) {
		return false
	}
	provided = strings.TrimPrefix(provided, prefix)
	return len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func writeHTTPJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

func writeHTTPError(writer http.ResponseWriter, status int, message string) {
	writeHTTPJSON(writer, status, map[string]string{"error": message})
}
