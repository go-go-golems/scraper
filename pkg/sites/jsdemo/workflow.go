package jsdemo

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/engine/model"
	storecontract "github.com/go-go-golems/scraper/pkg/engine/store"
)

type RunOptions struct {
	WorkflowID string
	Count      int
	Multiplier int
	Prefix     string
}

func (o RunOptions) normalize() RunOptions {
	ret := o
	if ret.Count <= 0 {
		ret.Count = 4
	}
	if ret.Multiplier == 0 {
		ret.Multiplier = 3
	}
	if ret.Prefix == "" {
		ret.Prefix = "demo"
	}
	if ret.WorkflowID == "" {
		ret.WorkflowID = fmt.Sprintf("js-demo-%d", time.Now().UTC().UnixNano())
	}
	return ret
}

func BuildWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	options = options.normalize()

	workflowID := model.WorkflowID(options.WorkflowID)
	seedID := model.OpID(options.WorkflowID + ":seed")
	summaryID := model.OpID(string(seedID) + ":summary")

	input, err := json.Marshal(map[string]any{
		"runID":      options.WorkflowID,
		"count":      options.Count,
		"multiplier": options.Multiplier,
		"prefix":     options.Prefix,
	})
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal js-demo input: %w", err)
	}

	return storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:    workflowID,
			Site:  model.SiteName("js-demo"),
			Name:  "js-demo workflow",
			Input: input,
		},
		Initial: []model.OpSpec{
			{
				ID:         seedID,
				WorkflowID: workflowID,
				Site:       model.SiteName("js-demo"),
				Kind:       "js",
				Queue:      model.QueueKey("site:js-demo:js"),
				DedupKey:   "js-demo:" + options.WorkflowID,
				Input:      input,
				Metadata:   map[string]string{"script": "seed.js"},
			},
		},
	}, summaryID, nil
}
