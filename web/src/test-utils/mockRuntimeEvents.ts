/**
 * Shared mock data factories for runtime events.
 * Used by Storybook stories and test utilities.
 */

import type { RuntimeEventV1 } from '../pb/proto/scraper/runtime/v1/events_pb';
import {
  RuntimeEventKind,
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../pb/proto/scraper/runtime/v1/events_pb';

let nextId = 0;

/**
 * Reset the auto-incrementing ID counter (call at the start of each story).
 */
export function resetMockIds(): void {
  nextId = 0;
}

/**
 * Create a single mock RuntimeEventV1 with optional overrides.
 */
export function mockEvent(overrides: Partial<RuntimeEventV1> = {}): RuntimeEventV1 {
  const id = `evt-${++nextId}`;
  const base: RuntimeEventV1 = {
    id,
    occurredAt: {
      seconds: BigInt(Math.floor(Date.now() / 1000) - nextId * 30),
      nanos: 0,
    },
    source: RuntimeEventSource.WORKER,
    severity: RuntimeEventSeverity.INFO,
    kind: RuntimeEventKind.OP_SUCCEEDED,
    message: `Event ${id}`,
    workflowId: 'wf-test-001',
    opId: `op-${nextId}`,
    site: 'hackernews',
    workerId: 'worker-01',
    payload: {},
  } as RuntimeEventV1;
  return { ...base, ...overrides };
}

const SEVERITIES = [
  RuntimeEventSeverity.DEBUG,
  RuntimeEventSeverity.INFO,
  RuntimeEventSeverity.WARN,
  RuntimeEventSeverity.ERROR,
];

const SOURCES = [
  RuntimeEventSource.SCHEDULER,
  RuntimeEventSource.WORKER,
  RuntimeEventSource.RUNNER,
  RuntimeEventSource.SERVER,
];

const KINDS = [
  RuntimeEventKind.WORKFLOW_CREATED,
  RuntimeEventKind.OP_LEASED,
  RuntimeEventKind.OP_SUCCEEDED,
  RuntimeEventKind.OP_FAILED,
  RuntimeEventKind.LOG_LINE,
];

const MESSAGES = [
  'Workflow started successfully',
  'Op dispatched to worker-01',
  'Operation completed in 342ms',
  'Connection timeout exceeded',
  'HTTP 503 from upstream server',
  'Retry attempt 2/3',
  'Artifact stored: page-1.html',
  'Token bucket refreshed',
  'Queue drained, no pending ops',
  'Script execution completed',
];

/**
 * Generate N mock events with varied severity, source, kind, and messages.
 */
export function generateMockEvents(count: number): RuntimeEventV1[] {
  resetMockIds();
  return Array.from({ length: count }, (_, i) =>
    mockEvent({
      source: SOURCES[i % SOURCES.length],
      severity: SEVERITIES[i % SEVERITIES.length],
      kind: KINDS[i % KINDS.length],
      message: MESSAGES[i % MESSAGES.length],
      payload: i % 3 === 0 ? { durationMillis: 100 + i * 50, attempt: 1 } : {},
    }),
  );
}
