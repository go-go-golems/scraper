// Package workflowv3 defines the canonical workflow-v3 IR, exact task and
// implementation identities, deterministic compiler, immutable task bundles,
// sealed worker registries, typed failures, and compact artifact references.
//
// It contains no scheduler or JavaScript-authoring state. Both direct Go plans
// and require("workflow") scripts compile through these representations.
package workflowv3
