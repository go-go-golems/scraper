package handlers

import (
	"net/http"

	runtimev1 "github.com/go-go-golems/scraper/gen/proto/scraper/runtime/v1"
	apitypes "github.com/go-go-golems/scraper/pkg/api/types"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/runtimeevents"
	"github.com/go-go-golems/scraper/pkg/services/submission"
)

type SubmissionHandler struct {
	service  *submission.Service
	engineDB string
	sitesDir string
	events   *runtimeevents.Publisher
}

func NewSubmissionHandler(service *submission.Service, engineDB string, sitesDir string, events *runtimeevents.Publisher) *SubmissionHandler {
	return &SubmissionHandler{
		service:  service,
		engineDB: engineDB,
		sitesDir: sitesDir,
		events:   events,
	}
}

func (h *SubmissionHandler) Submit(w http.ResponseWriter, r *http.Request) {
	request := apitypes.SubmitRequest{}
	if r.Body != nil {
		if err := decodeJSON(r, &request); err != nil {
			_ = runtimeevents.EmitSimpleEvent(h.events, &runtimev1.RuntimeEventV1{
				Source:    runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_SERVER,
				Component: "submission-handler",
				Kind:      runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_SUBMISSION_REJECTED,
				Severity:  runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_WARN,
				Message:   "submission rejected: invalid json",
				Site:      r.PathValue("site"),
				RequestId: runtimeevents.RequestIDFromContext(r.Context()),
				Payload: runtimeevents.BuildPayload(map[string]any{
					"verb":  r.PathValue("verb"),
					"error": err.Error(),
				}),
			})
			writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}

	response, err := h.service.Submit(r.Context(), submission.Request{
		Site:       model.SiteName(r.PathValue("site")),
		Verb:       r.PathValue("verb"),
		WorkflowID: request.WorkflowID,
		EngineDB:   h.engineDB,
		SitesDir:   h.sitesDir,
		Values:     request.Values,
		Sections:   request.Sections,
	})
	if err != nil {
		_ = runtimeevents.EmitSimpleEvent(h.events, &runtimev1.RuntimeEventV1{
			Source:    runtimev1.RuntimeEventSource_RUNTIME_EVENT_SOURCE_SERVER,
			Component: "submission-handler",
			Kind:      runtimev1.RuntimeEventKind_RUNTIME_EVENT_KIND_SUBMISSION_REJECTED,
			Severity:  runtimev1.RuntimeEventSeverity_RUNTIME_EVENT_SEVERITY_WARN,
			Message:   "submission rejected",
			Site:      r.PathValue("site"),
			RequestId: runtimeevents.RequestIDFromContext(r.Context()),
			Payload: runtimeevents.BuildPayload(map[string]any{
				"verb":  r.PathValue("verb"),
				"error": err.Error(),
			}),
		})
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, apitypes.SubmitResponse{
		Site:           response.Site,
		Verb:           response.Verb,
		CommandPath:    response.CommandPath,
		Workflow:       response.Result.Workflow,
		SiteDBPath:     response.Result.SiteDBPath,
		TargetOpID:     response.Result.TargetOpID,
		SubmittedCount: len(response.Result.Submitted),
		Submitted:      response.Result.Submitted,
		VerbData:       response.Result.VerbData,
	})
}
