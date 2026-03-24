package handlers

import (
	"net/http"
	"strconv"

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

func (h *EngineHandler) Workflows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	opts := engineview.ListWorkflowsOptions{
		Site:   model.SiteName(q.Get("site")),
		Status: model.WorkflowStatus(q.Get("status")),
		Limit:  limit,
		Offset: offset,
	}
	result, err := h.service.ListWorkflows(r.Context(), opts)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apitypes.WorkflowListResponse{
		Workflows: result.Workflows,
		Total:     result.Total,
	})
}

func (h *EngineHandler) Queues(w http.ResponseWriter, r *http.Request) {
	queues, err := h.service.ListQueues(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apitypes.QueueListResponse{Queues: queues})
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

func (h *EngineHandler) OpArtifacts(w http.ResponseWriter, r *http.Request) {
	workflowID := model.WorkflowID(r.PathValue("workflowID"))
	opID := model.OpID(r.PathValue("opID"))
	artifacts, err := h.service.ListArtifacts(r.Context(), workflowID, opID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, apitypes.ArtifactListResponse{Artifacts: artifacts})
}

func (h *EngineHandler) ArtifactDownload(w http.ResponseWriter, r *http.Request) {
	artifactID := model.ArtifactID(r.PathValue("artifactID"))
	artifact, err := h.service.GetArtifact(r.Context(), artifactID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if artifact == nil {
		writeError(w, http.StatusNotFound, "not_found", "artifact not found")
		return
	}
	ct := artifact.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Disposition", "inline; filename=\""+artifact.Name+"\"")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(artifact.Body)
}

func (h *EngineHandler) RetryOp(w http.ResponseWriter, r *http.Request) {
	workflowID := model.WorkflowID(r.PathValue("workflowID"))
	opID := model.OpID(r.PathValue("opID"))
	if err := h.service.RetryOp(r.Context(), workflowID, opID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "opID": opID})
}

func (h *EngineHandler) CancelWorkflow(w http.ResponseWriter, r *http.Request) {
	workflowID := model.WorkflowID(r.PathValue("workflowID"))
	if err := h.service.CancelWorkflow(r.Context(), workflowID); err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "workflowID": workflowID})
}
