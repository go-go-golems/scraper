package workflowv3sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-go-golems/scraper/pkg/workflowv3"
)

type ExpansionCandidate struct {
	RunID  workflowv3.RunID
	MapKey string
	Source workflowv3.ArtifactRef
}

type ChainedExpansionCandidate struct {
	RunID    workflowv3.RunID
	MapKey   string
	Manifest workflowv3.ItemManifest
	Final    bool
}

type ExpansionFinalizationCandidate struct {
	RunID  workflowv3.RunID
	MapKey string
}

type ExpansionPage struct {
	RunID      workflowv3.RunID
	MapKey     string
	Page       int
	FirstIndex int
	ItemCount  int
	NextIndex  int
	TotalItems int
	Expanded   bool
	PageDigest string
}

func (s *Store) ExpansionCandidates(ctx context.Context) ([]ExpansionCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT e.run_id, e.map_key, e.source_schema, e.source_digest,
  e.source_media_type, e.source_size_bytes, e.source_locator
FROM v3_expansions e JOIN v3_runs r ON r.run_id = e.run_id
WHERE r.status = 'running' AND e.status IN ('pending','expanding')
  AND e.source_digest IS NOT NULL
ORDER BY r.created_at, e.map_key`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var candidates []ExpansionCandidate
	for rows.Next() {
		var candidate ExpansionCandidate
		if err := rows.Scan(
			&candidate.RunID, &candidate.MapKey, &candidate.Source.Schema,
			&candidate.Source.Digest, &candidate.Source.MediaType,
			&candidate.Source.Size, &candidate.Source.Locator,
		); err != nil {
			return nil, err
		}
		_, mapped, _, err := s.loadPlanMap(ctx, candidate.RunID, candidate.MapKey)
		if err != nil {
			return nil, err
		}
		if mapped.Source.Source == "set-input" {
			candidates = append(candidates, candidate)
		}
	}
	return candidates, rows.Err()
}

func (s *Store) ChainedExpansionCandidate(ctx context.Context) (*ChainedExpansionCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT e.run_id, e.map_key, e.next_index, e.page_size
FROM v3_expansions e JOIN v3_runs r ON r.run_id = e.run_id
WHERE r.status = 'running' AND e.status IN ('pending','expanding')
ORDER BY r.created_at, e.map_key`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	type expansion struct {
		runID      workflowv3.RunID
		mapKey     string
		next, page int
	}
	var expansions []expansion
	for rows.Next() {
		var item expansion
		if err := rows.Scan(&item.runID, &item.mapKey, &item.next, &item.page); err != nil {
			return nil, err
		}
		expansions = append(expansions, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for _, candidate := range expansions {
		_, mapped, _, err := s.loadPlanMap(ctx, candidate.runID, candidate.mapKey)
		if err != nil {
			return nil, err
		}
		if mapped.Source.Source != "map-output" {
			continue
		}
		var upstreamStatus string
		var upstreamTotal int
		if err := s.db.QueryRowContext(ctx, `SELECT status, total_items FROM v3_expansions WHERE run_id = ? AND map_key = ?`, candidate.runID, mapped.Source.MapKey).Scan(&upstreamStatus, &upstreamTotal); err != nil {
			return nil, err
		}
		itemRows, err := s.db.QueryContext(ctx, `
SELECT item.item_key, node.status, output.schema_id, output.digest, output.media_type,
  output.size_bytes, output.locator
FROM v3_map_items item
JOIN v3_nodes node ON node.run_id = item.run_id AND node.node_key = item.node_key
LEFT JOIN v3_node_outputs output ON output.run_id = item.run_id AND output.node_key = item.node_key
WHERE item.run_id = ? AND item.map_key = ? ORDER BY item.item_index`, candidate.runID, mapped.Source.MapKey)
		if err != nil {
			return nil, err
		}
		items := []workflowv3.ManifestItem{}
		for itemRows.Next() {
			var key, status string
			var schema, digest, media, locator sql.NullString
			var size sql.NullInt64
			if err := itemRows.Scan(&key, &status, &schema, &digest, &media, &size, &locator); err != nil {
				_ = itemRows.Close()
				return nil, err
			}
			if status != "succeeded" || !schema.Valid || !digest.Valid || !media.Valid || !size.Valid || !locator.Valid {
				break
			}
			items = append(items, workflowv3.ManifestItem{Key: key, Value: workflowv3.ArtifactRef{Schema: schema.String, Digest: digest.String, MediaType: media.String, Size: size.Int64, Locator: locator.String}})
		}
		if err := itemRows.Close(); err != nil {
			return nil, err
		}
		final := upstreamStatus == "published" && len(items) == upstreamTotal
		if len(items)-candidate.next < candidate.page && !final {
			continue
		}
		manifest, err := workflowv3.NewItemManifest(mapped.Source.ItemSchema, items)
		if err != nil {
			return nil, err
		}
		return &ChainedExpansionCandidate{RunID: candidate.runID, MapKey: candidate.mapKey, Manifest: manifest, Final: final}, nil
	}
	return nil, nil
}

func (s *Store) ExpansionFinalizationCandidates(ctx context.Context) ([]ExpansionFinalizationCandidate, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT e.run_id, e.map_key
FROM v3_expansions e JOIN v3_runs r ON r.run_id = e.run_id
WHERE r.status = 'running' AND e.status = 'succeeded'
ORDER BY r.created_at, e.map_key`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var candidates []ExpansionFinalizationCandidate
	for rows.Next() {
		var candidate ExpansionFinalizationCandidate
		if err := rows.Scan(&candidate.RunID, &candidate.MapKey); err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

// ExpandNextPage materializes at most one deterministic bounded map page. A nil
// page means expansion is intentionally backpressured.
func (s *Store) ExpandNextPage(ctx context.Context, runID workflowv3.RunID, mapKey string, manifestRef workflowv3.ArtifactRef, manifest workflowv3.ItemManifest, now time.Time) (*ExpansionPage, error) {
	return s.expandNextPage(ctx, runID, mapKey, manifestRef, manifest, true, false, now)
}

func (s *Store) ExpandNextChainedPage(ctx context.Context, runID workflowv3.RunID, mapKey string, manifestRef workflowv3.ArtifactRef, manifest workflowv3.ItemManifest, final bool, now time.Time) (*ExpansionPage, error) {
	return s.expandNextPage(ctx, runID, mapKey, manifestRef, manifest, final, true, now)
}

func (s *Store) expandNextPage(
	ctx context.Context,
	runID workflowv3.RunID,
	mapKey string,
	manifestRef workflowv3.ArtifactRef,
	manifest workflowv3.ItemManifest,
	final, chained bool,
	now time.Time,
) (*ExpansionPage, error) {
	if err := workflowv3.ValidateArtifactRef(manifestRef); err != nil {
		return nil, fmt.Errorf("map manifest ref: %w", err)
	}
	body, err := workflowv3.EncodeItemManifest(manifest)
	if err != nil {
		return nil, fmt.Errorf("map manifest: %w", err)
	}
	digest, err := workflowv3.Digest(manifest)
	if err != nil {
		return nil, err
	}
	if manifestRef.Schema != workflowv3.ItemManifestSchemaV1 ||
		manifestRef.Digest != digest || manifestRef.Size != int64(len(body)) {
		return nil, fmt.Errorf("map manifest ref does not match canonical manifest")
	}

	plan, mapped, mapOrdinal, err := s.loadPlanMap(ctx, runID, mapKey)
	if err != nil {
		return nil, err
	}
	if (!chained && mapped.Source.Source != "set-input") || (chained && mapped.Source.Source != "map-output") {
		return nil, fmt.Errorf("map %q source is not available for this expansion mode", mapKey)
	}
	if manifest.ItemSchema != mapped.Source.ItemSchema {
		return nil, fmt.Errorf("map %q item schema %q does not match %q", mapKey, manifest.ItemSchema, mapped.Source.ItemSchema)
	}
	if len(manifest.Items) > mapped.Policy.MaxItems {
		return nil, fmt.Errorf("map %q cardinality %d exceeds %d", mapKey, len(manifest.Items), mapped.Policy.MaxItems)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var runStatus, status string
	var sourceSchema, sourceDigest, sourceMediaType, sourceLocator sql.NullString
	var sourceSize sql.NullInt64
	var totalItems, nextIndex, materialized, terminal int
	err = tx.QueryRowContext(ctx, `
SELECT r.status, e.status, e.source_schema, e.source_digest,
  e.source_media_type, e.source_size_bytes, e.source_locator,
  e.total_items, e.next_index, e.materialized_items, e.terminal_items
FROM v3_expansions e JOIN v3_runs r ON r.run_id = e.run_id
WHERE e.run_id = ? AND e.map_key = ?`, runID, mapKey).Scan(
		&runStatus, &status, &sourceSchema, &sourceDigest, &sourceMediaType,
		&sourceSize, &sourceLocator, &totalItems, &nextIndex, &materialized, &terminal,
	)
	if err != nil {
		return nil, err
	}
	if runStatus != "running" || status == "failed" || status == "canceled" || status == "succeeded" {
		return nil, fmt.Errorf("map %s/%s is not expandable", runID, mapKey)
	}
	if chained {
		if len(manifest.Items) < nextIndex {
			return nil, fmt.Errorf("map %q chained cardinality regressed", mapKey)
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_expansions SET source_schema = ?, source_digest = ?, source_media_type = ?,
  source_size_bytes = ?, source_locator = ?, updated_at = ?
WHERE run_id = ? AND map_key = ?`, manifestRef.Schema, manifestRef.Digest,
			manifestRef.MediaType, manifestRef.Size, manifestRef.Locator, formatTime(now), runID, mapKey); err != nil {
			return nil, err
		}
		totalItems = len(manifest.Items)
	} else {
		if !sourceSchema.Valid || sourceSchema.String != manifestRef.Schema ||
			!sourceDigest.Valid || sourceDigest.String != manifestRef.Digest ||
			!sourceMediaType.Valid || sourceMediaType.String != manifestRef.MediaType ||
			!sourceSize.Valid || sourceSize.Int64 != manifestRef.Size ||
			!sourceLocator.Valid || sourceLocator.String != manifestRef.Locator {
			return nil, fmt.Errorf("map %q source ref does not match submitted input", mapKey)
		}
		if totalItems >= 0 && totalItems != len(manifest.Items) {
			return nil, fmt.Errorf("map %q cardinality changed from %d to %d", mapKey, totalItems, len(manifest.Items))
		}
		if totalItems < 0 {
			totalItems = len(manifest.Items)
		}
	}
	if nextIndex == totalItems {
		if chained && !final {
			if err := tx.Commit(); err != nil {
				return nil, err
			}
			return nil, nil
		}
		terminalStatus := "expanded"
		if totalItems == 0 {
			terminalStatus = "succeeded"
		}
		if status != terminalStatus {
			if _, err := tx.ExecContext(ctx, `
UPDATE v3_expansions SET total_items = ?, status = ?, updated_at = ?
WHERE run_id = ? AND map_key = ? AND next_index = ?`,
				totalItems, terminalStatus, formatTime(now), runID, mapKey, nextIndex); err != nil {
				return nil, err
			}
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return &ExpansionPage{
			RunID: runID, MapKey: mapKey, Page: nextIndex / mapped.Policy.PageSize,
			FirstIndex: nextIndex, ItemCount: 0, NextIndex: nextIndex,
			TotalItems: totalItems, Expanded: true,
		}, nil
	}

	count := mapped.Policy.PageSize
	if remaining := totalItems - nextIndex; remaining < count {
		if chained && !final {
			if err := tx.Commit(); err != nil {
				return nil, err
			}
			return nil, nil
		}
		count = remaining
	}
	if mapped.Policy.MaxMaterializedAhead-(materialized-terminal) < count {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return nil, nil
	}
	items := manifest.Items[nextIndex : nextIndex+count]
	keys := make([]string, len(items))
	for i, item := range items {
		keys[i] = item.Key
	}
	identitySourceDigest := manifestRef.Digest
	if chained {
		identitySourceDigest, err = workflowv3.Digest(struct {
			Source string `json:"source"`
			MapKey string `json:"mapKey"`
		}{Source: mapped.Source.Source, MapKey: mapped.Source.MapKey})
		if err != nil {
			return nil, err
		}
	}
	pageDigest, err := workflowv3.Digest(struct {
		MapKey       string   `json:"mapKey"`
		SourceDigest string   `json:"sourceDigest"`
		FirstIndex   int      `json:"firstIndex"`
		ItemKeys     []string `json:"itemKeys"`
	}{MapKey: mapKey, SourceDigest: identitySourceDigest, FirstIndex: nextIndex, ItemKeys: keys})
	if err != nil {
		return nil, err
	}
	pageNo := nextIndex / mapped.Policy.PageSize
	ordinalBase := len(plan.Nodes)
	for index := 0; index < mapOrdinal; index++ {
		ordinalBase += plan.Maps[index].Policy.MaxItems
	}
	for offset, item := range items {
		itemIndex := nextIndex + offset
		nodeKey, err := workflowv3.MapChildNodeKey(mapKey, identitySourceDigest, item.Key)
		if err != nil {
			return nil, err
		}
		bindings, err := mapItemBindings(mapped.Bindings, mapKey)
		if err != nil {
			return nil, err
		}
		bindingsBody, _ := workflowv3.CanonicalJSON(bindings)
		inputSchemas, _ := workflowv3.CanonicalJSON(mapped.InputSchemas)
		outputSchemas, _ := workflowv3.CanonicalJSON(mapped.OutputSchemas)
		modules, _ := workflowv3.CanonicalJSON(mapped.Modules)
		identity := mapped.Implementation
		isolation := workflowv3.EffectivePlanIsolation(mapped.Isolation)
		isolationBody, _ := workflowv3.CanonicalJSON(isolation)
		var budgetAccount, budgetOnExhausted, budgetApprovalGate any
		if mapped.Budget != nil {
			budgetAccount, budgetOnExhausted = mapped.Budget.Account, mapped.Budget.OnExhausted
			if mapped.Budget.ApprovalGate != "" {
				budgetApprovalGate = mapped.Budget.ApprovalGate
			}
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_nodes(
  run_id, node_key, ordinal, task_kind, task_version, bundle_digest,
  entrypoint, task_abi, bindings_json, input_schemas_json,
  output_schemas_json, modules_json, resource_class, max_attempts,
  retry_backoff_ms, budget_account, budget_on_exhausted,
  budget_approval_gate, isolation_class, isolation_policy_digest,
  isolation_executor_digest, isolation_policy_json, status
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending')`,
			runID, nodeKey, ordinalBase+itemIndex, identity.Kind, identity.Version,
			identity.BundleDigest, identity.Entrypoint, identity.ABI,
			bindingsBody, inputSchemas, outputSchemas, modules, mapped.ResourceClass,
			mapped.Retry.MaxAttempts, mapped.Retry.BackoffMillis,
			budgetAccount, budgetOnExhausted, budgetApprovalGate,
			isolation.Effective.Class, isolation.PolicyDigest, isolation.ExecutorDigest, isolationBody); err != nil {
			return nil, fmt.Errorf("insert map child %s: %w", item.Key, err)
		}
		if err := insertNodeBudget(ctx, tx, runID, nodeKey, mapped.Budget); err != nil {
			return nil, err
		}
		if err := insertRef(ctx, tx, `
INSERT INTO v3_map_items(
  run_id, map_key, item_key, item_index, node_key, input_schema,
  input_digest, input_media_type, input_size_bytes, input_locator
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			[]any{runID, mapKey, item.Key, itemIndex, nodeKey}, item.Value); err != nil {
			return nil, fmt.Errorf("insert map item %s: %w", item.Key, err)
		}
		if err := insertNodeReadiness(ctx, tx, runID, nodeKey, bindings, nil); err != nil {
			return nil, err
		}
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO v3_expansion_pages(
  run_id, map_key, page_no, first_index, item_count, page_digest, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		runID, mapKey, pageNo, nextIndex, count, pageDigest, formatTime(now)); err != nil {
		return nil, fmt.Errorf("insert expansion page: %w", err)
	}
	newNext := nextIndex + count
	newStatus := "expanding"
	if newNext == totalItems && final {
		newStatus = "expanded"
	}
	result, err := tx.ExecContext(ctx, `
UPDATE v3_expansions
SET total_items = ?, next_index = ?, materialized_items = materialized_items + ?,
  status = ?, updated_at = ?
WHERE run_id = ? AND map_key = ? AND next_index = ? AND status IN ('pending','expanding')`,
		totalItems, newNext, count, newStatus, formatTime(now), runID, mapKey, nextIndex)
	if err != nil {
		return nil, err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, fmt.Errorf("map expansion cursor changed")
	}
	if err := insertEvent(ctx, tx, runID, "", "map.page_materialized", map[string]any{
		"mapKey": mapKey, "page": pageNo, "firstIndex": nextIndex,
		"itemCount": count, "pageDigest": pageDigest,
	}, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &ExpansionPage{
		RunID: runID, MapKey: mapKey, Page: pageNo, FirstIndex: nextIndex,
		ItemCount: count, NextIndex: newNext, TotalItems: totalItems,
		Expanded: newStatus == "expanded", PageDigest: pageDigest,
	}, nil
}

func (s *Store) MapOutputManifest(ctx context.Context, runID workflowv3.RunID, mapKey string) (workflowv3.ItemManifest, error) {
	_, mapped, _, err := s.loadPlanMap(ctx, runID, mapKey)
	if err != nil {
		return workflowv3.ItemManifest{}, err
	}
	if len(mapped.OutputSchemas) != 1 {
		return workflowv3.ItemManifest{}, fmt.Errorf("map %q must have one output", mapKey)
	}
	var outputPort, outputSchema string
	for port, schema := range mapped.OutputSchemas {
		outputPort, outputSchema = port, schema
	}
	var status string
	var totalItems, terminalItems int
	if err := s.db.QueryRowContext(ctx, `
SELECT status, total_items, terminal_items FROM v3_expansions
WHERE run_id = ? AND map_key = ?`, runID, mapKey).Scan(&status, &totalItems, &terminalItems); err != nil {
		return workflowv3.ItemManifest{}, err
	}
	if status != "succeeded" || totalItems != terminalItems {
		return workflowv3.ItemManifest{}, fmt.Errorf("map %s/%s is not ready for publication", runID, mapKey)
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT item.item_key, output.schema_id, output.digest, output.media_type,
  output.size_bytes, output.locator
FROM v3_map_items item
JOIN v3_nodes node
  ON node.run_id = item.run_id AND node.node_key = item.node_key
JOIN v3_node_outputs output
  ON output.run_id = item.run_id AND output.node_key = item.node_key
WHERE item.run_id = ? AND item.map_key = ? AND node.status = 'succeeded'
  AND output.port = ?
ORDER BY item.item_index`, runID, mapKey, outputPort)
	if err != nil {
		return workflowv3.ItemManifest{}, err
	}
	defer func() { _ = rows.Close() }()
	items := make([]workflowv3.ManifestItem, 0, totalItems)
	for rows.Next() {
		var item workflowv3.ManifestItem
		if err := rows.Scan(
			&item.Key, &item.Value.Schema, &item.Value.Digest, &item.Value.MediaType,
			&item.Value.Size, &item.Value.Locator,
		); err != nil {
			return workflowv3.ItemManifest{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return workflowv3.ItemManifest{}, err
	}
	if len(items) != totalItems {
		return workflowv3.ItemManifest{}, fmt.Errorf("map %q output cardinality %d does not match %d", mapKey, len(items), totalItems)
	}
	return workflowv3.NewItemManifest(outputSchema, items)
}

func (s *Store) PublishMapOutput(
	ctx context.Context,
	runID workflowv3.RunID,
	mapKey string,
	output workflowv3.ArtifactRef,
	now time.Time,
) error {
	if err := workflowv3.ValidateArtifactRef(output); err != nil {
		return err
	}
	if output.Schema != workflowv3.ItemManifestSchemaV1 {
		return fmt.Errorf("map output must use schema %q", workflowv3.ItemManifestSchemaV1)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var status string
	var existing workflowv3.ArtifactRef
	var schema, digest, mediaType, locator sql.NullString
	var size sql.NullInt64
	if err := tx.QueryRowContext(ctx, `
SELECT status, output_schema, output_digest, output_media_type,
  output_size_bytes, output_locator
FROM v3_expansions WHERE run_id = ? AND map_key = ?`, runID, mapKey).Scan(
		&status, &schema, &digest, &mediaType, &size, &locator,
	); err != nil {
		return err
	}
	if status == "published" {
		if schema.Valid && digest.Valid && mediaType.Valid && size.Valid && locator.Valid {
			existing = workflowv3.ArtifactRef{
				Schema: schema.String, Digest: digest.String, MediaType: mediaType.String,
				Size: size.Int64, Locator: locator.String,
			}
		}
		if existing == output {
			return tx.Commit()
		}
		return fmt.Errorf("map %s/%s is already published with another output", runID, mapKey)
	}
	if status != "succeeded" {
		return fmt.Errorf("map %s/%s is not ready for publication", runID, mapKey)
	}
	result, err := tx.ExecContext(ctx, `
UPDATE v3_expansions SET status = 'published', output_schema = ?,
  output_digest = ?, output_media_type = ?, output_size_bytes = ?,
  output_locator = ?, updated_at = ?
WHERE run_id = ? AND map_key = ? AND status = 'succeeded'`,
		output.Schema, output.Digest, output.MediaType, output.Size, output.Locator,
		formatTime(now), runID, mapKey)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return fmt.Errorf("map publication state changed")
	}
	var planBody []byte
	if err := tx.QueryRowContext(ctx, `SELECT plan_json FROM v3_runs WHERE run_id = ?`, runID).Scan(&planBody); err != nil {
		return err
	}
	var plan workflowv3.WorkflowPlan
	if err := workflowv3.StrictDecode(planBody, &plan); err != nil {
		return err
	}
	for _, downstream := range plan.Maps {
		if downstream.Source.Source != "map-output" || downstream.Source.MapKey != mapKey {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
UPDATE v3_expansions SET source_schema = ?, source_digest = ?, source_media_type = ?,
  source_size_bytes = ?, source_locator = ?, updated_at = ?
WHERE run_id = ? AND map_key = ? AND status = 'pending' AND source_digest IS NULL`,
			output.Schema, output.Digest, output.MediaType, output.Size, output.Locator,
			formatTime(now), runID, downstream.Key); err != nil {
			return fmt.Errorf("activate downstream map %s: %w", downstream.Key, err)
		}
	}
	if err := insertEvent(ctx, tx, runID, "", "map.published", map[string]any{
		"mapKey": mapKey, "digest": output.Digest,
	}, now); err != nil {
		return err
	}
	if _, _, err := reconcileRunStateTx(ctx, tx, runID, now); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) loadPlanMap(ctx context.Context, runID workflowv3.RunID, mapKey string) (workflowv3.WorkflowPlan, workflowv3.PlanMap, int, error) {
	var body []byte
	if err := s.db.QueryRowContext(ctx, `SELECT plan_json FROM v3_runs WHERE run_id = ?`, runID).Scan(&body); err != nil {
		return workflowv3.WorkflowPlan{}, workflowv3.PlanMap{}, 0, err
	}
	var plan workflowv3.WorkflowPlan
	if err := workflowv3.StrictDecode(body, &plan); err != nil {
		return workflowv3.WorkflowPlan{}, workflowv3.PlanMap{}, 0, err
	}
	for index, mapped := range plan.Maps {
		if mapped.Key == mapKey {
			return plan, mapped, index, nil
		}
	}
	return workflowv3.WorkflowPlan{}, workflowv3.PlanMap{}, 0, fmt.Errorf("run %s has no map %q", runID, mapKey)
}

func mapItemBindings(bindings map[string]workflowv3.ValueRef, mapKey string) (map[string]workflowv3.ValueRef, error) {
	ret := make(map[string]workflowv3.ValueRef, len(bindings))
	itemBindings := 0
	for port, binding := range bindings {
		if binding.Source == "map-item" {
			if binding.MapKey != mapKey {
				return nil, fmt.Errorf("map item binding owner mismatch")
			}
			itemBindings++
		}
		ret[port] = binding
	}
	if itemBindings != 1 {
		return nil, fmt.Errorf("map requires exactly one item binding")
	}
	return ret, nil
}
