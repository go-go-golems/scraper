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
	Index      int
}

func (o RunOptions) normalize(defaultPrefix string) RunOptions {
	ret := o
	if ret.Count <= 0 {
		ret.Count = 4
	}
	if ret.Multiplier == 0 {
		ret.Multiplier = 3
	}
	if ret.Prefix == "" {
		ret.Prefix = defaultPrefix
	}
	if ret.Index < 0 {
		ret.Index = 0
	}
	return ret
}

func ensureWorkflowID(current string, kind string) string {
	if current != "" {
		return current
	}
	return fmt.Sprintf("js-demo-%s-%d", kind, time.Now().UTC().UnixNano())
}

func seedInput(options RunOptions) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"runID":      options.WorkflowID,
		"count":      options.Count,
		"multiplier": options.Multiplier,
		"prefix":     options.Prefix,
	})
}

func itemInput(options RunOptions) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"runID":      options.WorkflowID,
		"index":      options.Index,
		"multiplier": options.Multiplier,
		"prefix":     options.Prefix,
	})
}

func itemOpID(workflowID string, index int) model.OpID {
	return model.OpID(fmt.Sprintf("%s:item:%02d", workflowID, index+1))
}

func BuildWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	return BuildSeedWorkflow(options)
}

func BuildSeedWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	options = options.normalize("demo")
	options.WorkflowID = ensureWorkflowID(options.WorkflowID, "seed")

	workflowID := model.WorkflowID(options.WorkflowID)
	seedID := model.OpID(options.WorkflowID + ":seed")
	summaryID := model.OpID(string(seedID) + ":summary")

	input, err := seedInput(options)
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal js-demo input: %w", err)
	}

	return storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:    workflowID,
			Site:  model.SiteName("js-demo"),
			Name:  "js-demo seed workflow",
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

func BuildItemWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	options = options.normalize("item")
	options.WorkflowID = ensureWorkflowID(options.WorkflowID, "item")

	workflowID := model.WorkflowID(options.WorkflowID)
	itemID := itemOpID(options.WorkflowID, options.Index)
	input, err := itemInput(options)
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal js-demo item input: %w", err)
	}

	return storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:    workflowID,
			Site:  model.SiteName("js-demo"),
			Name:  "js-demo item workflow",
			Input: input,
		},
		Initial: []model.OpSpec{
			{
				ID:         itemID,
				WorkflowID: workflowID,
				Site:       model.SiteName("js-demo"),
				Kind:       "js",
				Queue:      model.QueueKey("site:js-demo:js"),
				DedupKey:   "js-demo:item:" + options.WorkflowID + ":" + fmt.Sprintf("%02d", options.Index+1),
				Input:      input,
				Metadata:   map[string]string{"script": "build_item.js"},
			},
		},
	}, itemID, nil
}

func BuildSummaryWorkflow(options RunOptions) (storecontract.CreateWorkflowParams, model.OpID, error) {
	options = options.normalize("summary")
	options.WorkflowID = ensureWorkflowID(options.WorkflowID, "summary")

	workflowID := model.WorkflowID(options.WorkflowID)
	ops := make([]model.OpSpec, 0, options.Count+1)
	itemOpIDs := make([]string, 0, options.Count)

	for i := 0; i < options.Count; i++ {
		itemID := itemOpID(options.WorkflowID, i)
		itemOpIDs = append(itemOpIDs, string(itemID))
		input, err := itemInput(RunOptions{
			WorkflowID: options.WorkflowID,
			Multiplier: options.Multiplier,
			Prefix:     options.Prefix,
			Index:      i,
		})
		if err != nil {
			return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal js-demo summary item input: %w", err)
		}

		ops = append(ops, model.OpSpec{
			ID:         itemID,
			WorkflowID: workflowID,
			Site:       model.SiteName("js-demo"),
			Kind:       "js",
			Queue:      model.QueueKey("site:js-demo:js"),
			DedupKey:   "js-demo:summary-item:" + options.WorkflowID + ":" + fmt.Sprintf("%02d", i+1),
			Input:      input,
			Metadata:   map[string]string{"script": "build_item.js"},
		})
	}

	summaryID := model.OpID(options.WorkflowID + ":summary")
	summaryInput, err := json.Marshal(map[string]any{
		"runID":     options.WorkflowID,
		"itemOpIDs": itemOpIDs,
	})
	if err != nil {
		return storecontract.CreateWorkflowParams{}, "", fmt.Errorf("marshal js-demo summary input: %w", err)
	}

	dependsOn := make([]model.Dependency, 0, len(itemOpIDs))
	for _, opID := range itemOpIDs {
		dependsOn = append(dependsOn, model.Dependency{
			OpID:     model.OpID(opID),
			Required: true,
		})
	}

	ops = append(ops, model.OpSpec{
		ID:         summaryID,
		WorkflowID: workflowID,
		Site:       model.SiteName("js-demo"),
		Kind:       "js",
		Queue:      model.QueueKey("site:js-demo:js"),
		DedupKey:   "js-demo:summary:" + options.WorkflowID,
		Input:      summaryInput,
		DependsOn:  dependsOn,
		Metadata:   map[string]string{"script": "summarize.js"},
	})

	return storecontract.CreateWorkflowParams{
		Workflow: model.WorkflowRun{
			ID:    workflowID,
			Site:  model.SiteName("js-demo"),
			Name:  "js-demo summary workflow",
			Input: summaryInput,
		},
		Initial: ops,
	}, summaryID, nil
}
