package types

import (
	"encoding/json"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	"github.com/go-go-golems/scraper/pkg/engine/store/sqlite"
	"github.com/go-go-golems/scraper/pkg/services/catalog"
	"github.com/go-go-golems/scraper/pkg/services/engineview"
)

type ErrorResponse struct {
	Error APIError `json:"error"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type InfoResponse struct {
	Version  string                `json:"version"`
	Address  string                `json:"address"`
	EngineDB string                `json:"engineDB"`
	SitesDir string                `json:"sitesDir"`
	Sites    []catalog.SiteSummary `json:"sites"`
	Now      time.Time             `json:"now"`
}

type SubmitRequest struct {
	WorkflowID string                            `json:"workflowID,omitempty"`
	Values     map[string]interface{}            `json:"values,omitempty"`
	Sections   map[string]map[string]interface{} `json:"sections,omitempty"`
}

type SubmitResponse struct {
	Site           model.SiteName    `json:"site"`
	Verb           string            `json:"verb"`
	CommandPath    string            `json:"commandPath"`
	Workflow       model.WorkflowRun `json:"workflow"`
	SiteDBPath     string            `json:"siteDBPath"`
	TargetOpID     model.OpID        `json:"targetOpID,omitempty"`
	SubmittedCount int               `json:"submittedCount"`
	Submitted      []model.OpSpec    `json:"submitted"`
	VerbData       json.RawMessage   `json:"verbData,omitempty"`
}

type EngineStatusResponse struct {
	Status *sqlite.EngineStatus `json:"status"`
}

type WorkflowListResponse struct {
	Workflows []engineview.WorkflowListItem `json:"workflows"`
	Total     int                           `json:"total"`
}

type QueueListResponse struct {
	Queues []engineview.QueueStatus `json:"queues"`
}

type ArtifactListResponse struct {
	Artifacts []engineview.ArtifactSummary `json:"artifacts"`
}

type ScriptListResponse struct {
	Scripts []string `json:"scripts"`
}

type ScriptResponse struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

type WorkflowResponse struct {
	Workflow *engineview.WorkflowSummary `json:"workflow"`
}

type WorkflowOpsResponse struct {
	WorkflowID model.WorkflowID        `json:"workflowID"`
	Ops        []engineview.WorkflowOp `json:"ops"`
}
