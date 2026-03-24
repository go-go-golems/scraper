package handlers

import (
	"net/http"

	apitypes "github.com/go-go-golems/scraper/pkg/api/types"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
)

type EngineHandler struct {
	service *engineview.Service
}

func NewEngineHandler(service *engineview.Service) *EngineHandler {
	return &EngineHandler{service: service}
}

func (h *EngineHandler) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.EngineStatus(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apitypes.EngineStatusResponse{Status: status})
}

func (h *EngineHandler) Migrations(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.EngineStatus(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"path":                 status.Path,
		"exists":               status.Exists,
		"initialized":          status.Initialized,
		"currentVersion":       status.CurrentVersion,
		"latestKnownMigration": status.LatestKnownMigration,
		"migrationsUpToDate":   status.MigrationsUpToDate,
		"migrations":           status.Migrations,
	})
}

func (h *EngineHandler) Workflow(w http.ResponseWriter, r *http.Request) {
	workflowID := model.WorkflowID(r.PathValue("workflowID"))
	workflow, err := h.service.Workflow(r.Context(), workflowID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if workflow == nil {
		writeError(w, http.StatusNotFound, "not_found", "workflow not found")
		return
	}
	writeJSON(w, http.StatusOK, apitypes.WorkflowResponse{Workflow: workflow})
}

func (h *EngineHandler) WorkflowOps(w http.ResponseWriter, r *http.Request) {
	workflowID := model.WorkflowID(r.PathValue("workflowID"))
	ops, err := h.service.WorkflowOps(r.Context(), workflowID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if ops == nil {
		writeError(w, http.StatusNotFound, "not_found", "workflow not found")
		return
	}
	writeJSON(w, http.StatusOK, apitypes.WorkflowOpsResponse{
		WorkflowID: workflowID,
		Ops:        ops,
	})
}
