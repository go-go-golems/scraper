// Package workflowv3sqlite persists compact workflow-v3 runs, nodes,
// append-only attempts, leases, output refs, and redacted events. Completion is
// transactionally fenced by lease token and cancellation epoch.
package workflowv3sqlite
