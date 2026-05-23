package engineview

import (
	"context"
	"fmt"

	"github.com/go-go-golems/scraper/pkg/engine/model"
)

type QueueStatus struct {
	Site        model.SiteName `json:"site"`
	Queue       model.QueueKey `json:"queue"`
	Pending     int            `json:"pending"`
	Ready       int            `json:"ready"`
	Running     int            `json:"running"`
	Succeeded   int            `json:"succeeded"`
	Failed      int            `json:"failed"`
	InFlight    int            `json:"inFlight"`
	MaxInFlight int            `json:"maxInFlight"`
	Tokens      *float64       `json:"tokens,omitempty"`
	Burst       *int           `json:"burst,omitempty"`
	RatePerSec  *float64       `json:"ratePerSecond,omitempty"`
}

func (s *Service) ListQueues(ctx context.Context) ([]QueueStatus, error) {
	db, err := s.openReadDB()
	if err != nil {
		return nil, err
	}
	if db == nil {
		return []QueueStatus{}, nil
	}
	defer func() { _ = db.Close() }()

	query := `SELECT o.site, o.queue_key, o.status, COUNT(1)
		FROM ops o
		GROUP BY o.site, o.queue_key, o.status
		ORDER BY o.site, o.queue_key, o.status`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list queue op counts: %w", err)
	}
	defer func() { _ = rows.Close() }()

	type queueKey struct {
		site  model.SiteName
		queue model.QueueKey
	}
	queueMap := map[queueKey]*QueueStatus{}
	var order []queueKey
	for rows.Next() {
		var site model.SiteName
		var queue model.QueueKey
		var status model.OpStatus
		var count int
		if err := rows.Scan(&site, &queue, &status, &count); err != nil {
			return nil, fmt.Errorf("scan queue status: %w", err)
		}
		key := queueKey{site, queue}
		qs, ok := queueMap[key]
		if !ok {
			qs = &QueueStatus{Site: site, Queue: queue, MaxInFlight: 1}
			queueMap[key] = qs
			order = append(order, key)
		}
		switch status {
		case model.OpStatusPending:
			qs.Pending = count
		case model.OpStatusReady:
			qs.Ready = count
		case model.OpStatusRunning:
			qs.Running = count
			qs.InFlight = count
		case model.OpStatusSucceeded:
			qs.Succeeded = count
		case model.OpStatusFailed:
			qs.Failed = count
		case model.OpStatusCanceled:
			// Canceled ops do not have a dedicated queue summary field yet.
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	tokenRows, err := db.QueryContext(ctx, `SELECT site, queue_key, tokens FROM queue_limit_state`)
	if err == nil {
		defer func() { _ = tokenRows.Close() }()
		for tokenRows.Next() {
			var site model.SiteName
			var queue model.QueueKey
			var tokens float64
			if err := tokenRows.Scan(&site, &queue, &tokens); err == nil {
				key := queueKey{site, queue}
				if qs, ok := queueMap[key]; ok {
					qs.Tokens = &tokens
				}
			}
		}
	}

	result := make([]QueueStatus, 0, len(order))
	for _, key := range order {
		result = append(result, *queueMap[key])
	}
	return result, nil
}
