package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpSpecSupportsWorkflowLinkage(t *testing.T) {
	parentID := OpID("op-parent")
	spec := OpSpec{
		ID:         OpID("op-child"),
		WorkflowID: WorkflowID("workflow-1"),
		ParentID:   &parentID,
		Site:       SiteName("nereval"),
		Kind:       "js/analyze",
		Queue:      QueueKey("site:nereval:http"),
		Input:      json.RawMessage(`{"page":2}`),
		DependsOn: []Dependency{
			{OpID: OpID("op-fetch-list"), Required: true},
		},
		Retry: RetryPolicy{
			MaxAttempts:    5,
			BackoffKind:    BackoffKindExponential,
			InitialBackoff: 2 * time.Second,
			MaxBackoff:     1 * time.Minute,
			Multiplier:     2,
		},
	}

	require.Equal(t, OpID("op-parent"), *spec.ParentID)
	require.Len(t, spec.DependsOn, 1)
	require.Equal(t, BackoffKindExponential, spec.Retry.BackoffKind)
}

func TestOpResultTracksEmittedOpIDsSeparately(t *testing.T) {
	result := OpResult{
		OpID:       OpID("op-list-extract"),
		Emitted:    []OpSpec{{ID: OpID("op-detail-1")}, {ID: OpID("op-detail-2")}},
		EmittedIDs: []OpID{OpID("op-detail-1"), OpID("op-detail-2")},
	}

	require.Len(t, result.Emitted, 2)
	require.Equal(t, []OpID{OpID("op-detail-1"), OpID("op-detail-2")}, result.EmittedIDs)
}
