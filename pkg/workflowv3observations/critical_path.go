package workflowv3observations

import (
	"sort"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type criticalPathEntry struct {
	NodeKey          workflowv3.NodeKey `json:"nodeKey"`
	Predecessor      workflowv3.NodeKey `json:"predecessor,omitempty"`
	AttemptCount     int                `json:"attemptCount"`
	ExecutionMicros  int64              `json:"executionMicros"`
	CumulativeMicros int64              `json:"cumulativeMicros"`
}

func criticalPath(source SourceSnapshot, limit int) (Trace, CountCoverage) {
	attempts := map[workflowv3.NodeKey][]AttemptSource{}
	for _, attempt := range source.Attempts {
		attempts[attempt.NodeKey] = append(attempts[attempt.NodeKey], attempt)
	}
	nodes := make([]NodeSource, 0, len(source.Nodes))
	for _, node := range source.Nodes {
		if node.Origin == "static" {
			nodes = append(nodes, node)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].NodeKey < nodes[j].NodeKey })
	byKey := map[workflowv3.NodeKey]NodeSource{}
	for _, node := range nodes {
		byKey[node.NodeKey] = node
	}
	done := map[workflowv3.NodeKey]criticalPathEntry{}
	remaining := append([]NodeSource(nil), nodes...)
	for len(remaining) > 0 {
		progress := false
		next := remaining[:0]
		for _, node := range remaining {
			ready := true
			for _, dependency := range node.Dependencies {
				if _, known := byKey[dependency]; known {
					if _, complete := done[dependency]; !complete {
						ready = false
						break
					}
				}
			}
			if !ready {
				next = append(next, node)
				continue
			}
			weight := int64(0)
			for _, attempt := range attempts[node.NodeKey] {
				weight += attempt.FinishedAt.Sub(attempt.StartedAt).Microseconds()
			}
			entry := criticalPathEntry{NodeKey: node.NodeKey, AttemptCount: len(attempts[node.NodeKey]), ExecutionMicros: weight, CumulativeMicros: weight}
			for _, dependency := range node.Dependencies {
				candidate, ok := done[dependency]
				if ok && candidate.CumulativeMicros+weight > entry.CumulativeMicros {
					entry.Predecessor = dependency
					entry.CumulativeMicros = candidate.CumulativeMicros + weight
				}
			}
			done[node.NodeKey] = entry
			progress = true
		}
		remaining = next
		if !progress {
			break
		}
	}
	var terminal criticalPathEntry
	for _, current := range done {
		if current.CumulativeMicros > terminal.CumulativeMicros || (current.CumulativeMicros == terminal.CumulativeMicros && current.NodeKey < terminal.NodeKey) {
			terminal = current
		}
	}
	var reverse []criticalPathEntry
	for terminal.NodeKey != "" {
		reverse = append(reverse, terminal)
		if terminal.Predecessor == "" {
			break
		}
		terminal = done[terminal.Predecessor]
	}
	path := make([]criticalPathEntry, len(reverse))
	for index := range reverse {
		path[len(reverse)-1-index] = reverse[index]
	}
	truncated := len(path) > limit
	if truncated {
		path = path[len(path)-limit:]
	}
	value := mustJSON(map[string]any{"boundary": "dependency-weighted-closed-attempt-execution/v1", "entries": path})
	return Trace{Kind: "workflow.critical_path", SchemaVersion: "scraper-workflow-critical-path/v1", Value: value, Truncated: truncated}, CountCoverage{Observed: len(done), Total: len(source.Nodes)}
}
