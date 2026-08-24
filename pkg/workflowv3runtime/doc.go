// Package workflowv3runtime executes exact workflow-v3 plan nodes through a
// sealed registry. Every attempt receives a fresh Goja runtime, a lease-scoped
// task context, only its declared host modules, and validated artifact refs.
package workflowv3runtime
