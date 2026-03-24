package handlers

import (
	"net/http"

	apitypes "github.com/go-go-golems/scraper/pkg/api/types"
	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/services/submission"
)

type SubmissionHandler struct {
	service  *submission.Service
	engineDB string
	sitesDir string
}

func NewSubmissionHandler(service *submission.Service, engineDB string, sitesDir string) *SubmissionHandler {
	return &SubmissionHandler{
		service:  service,
		engineDB: engineDB,
		sitesDir: sitesDir,
	}
}

func (h *SubmissionHandler) Submit(w http.ResponseWriter, r *http.Request) {
	request := apitypes.SubmitRequest{}
	if r.Body != nil {
		if err := decodeJSON(r, &request); err != nil {
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
