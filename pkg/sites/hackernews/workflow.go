package hackernews

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
)

type RunOptions struct {
	WorkflowID string
	BaseURL    string
}

func normalizeRunOptions(options RunOptions) RunOptions {
	ret := options
	if ret.BaseURL == "" {
		ret.BaseURL = "https://news.ycombinator.com/"
	}
	return ret
}

func ensureWorkflowID(current string, kind string) string {
	if current != "" {
		return current
	}
	return fmt.Sprintf("hackernews-%s-%d", kind, time.Now().UTC().UnixNano())
}

func seedInput(baseURL string) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"baseURL": baseURL,
	})
}

func BuildSeedWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	options = normalizeRunOptions(options)
	options.WorkflowID = ensureWorkflowID(options.WorkflowID, "seed")

	input, err := seedInput(options.BaseURL)
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal hackernews seed input: %w", err)
	}

	workflowID := model.WorkflowID(options.WorkflowID)
	seedID := model.OpID(options.WorkflowID + ":seed")
	targetOpID := model.OpID(string(seedID) + ":frontpage-extract")

	return storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:    workflowID,
			Site:  model.SiteName("hackernews"),
			Name:  "hackernews seed workflow",
			Input: input,
		},
		Initial: []model.OpSpec{
			{
				ID:         seedID,
				WorkflowID: workflowID,
				Site:       model.SiteName("hackernews"),
				Kind:       "js",
				Queue:      model.QueueKey("site:hackernews:js"),
				DedupKey:   "hackernews:seed:" + options.BaseURL,
				Input:      input,
				Metadata:   map[string]string{"script": "seed.js"},
			},
		},
	}, targetOpID, nil
}

func BuildExtractFrontpageWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	options = normalizeRunOptions(options)
	options.WorkflowID = ensureWorkflowID(options.WorkflowID, "extract-frontpage")

	workflowID := model.WorkflowID(options.WorkflowID)
	fetchID := model.OpID(options.WorkflowID + ":frontpage-fetch")
	extractID := model.OpID(options.WorkflowID + ":frontpage-extract")

	fetchInput, err := json.Marshal(map[string]any{
		"request": map[string]any{
			"method": "GET",
			"url":    options.BaseURL,
		},
		"persistBody":  true,
		"artifactName": "frontpage.html",
	})
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal hackernews fetch input: %w", err)
	}
	extractInput, err := json.Marshal(map[string]any{
		"baseURL":     options.BaseURL,
		"fetchedOpID": fetchID,
	})
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal hackernews extract input: %w", err)
	}

	return storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:   workflowID,
			Site: model.SiteName("hackernews"),
			Name: "hackernews extract workflow",
		},
		Initial: []model.OpSpec{
			{
				ID:         fetchID,
				WorkflowID: workflowID,
				Site:       model.SiteName("hackernews"),
				Kind:       "http/fetch",
				Queue:      model.QueueKey("site:hackernews:http"),
				DedupKey:   "hackernews:frontpage:" + options.BaseURL,
				Input:      fetchInput,
			},
			{
				ID:         extractID,
				WorkflowID: workflowID,
				Site:       model.SiteName("hackernews"),
				Kind:       "js",
				Queue:      model.QueueKey("site:hackernews:js"),
				DedupKey:   "hackernews:frontpage-extract:" + options.BaseURL,
				Input:      extractInput,
				DependsOn: []model.Dependency{
					{
						OpID:     fetchID,
						Required: true,
					},
				},
				Metadata: map[string]string{"script": "extract_frontpage.js"},
			},
		},
	}, extractID, nil
}
