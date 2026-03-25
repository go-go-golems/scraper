import { describe, expect, it } from 'vitest';
import { create } from '@bufbuild/protobuf';
import { TimestampSchema } from '@bufbuild/protobuf/wkt';
import {
  RuntimeEventSeverity,
  RuntimeEventSource,
  type RuntimeEventV1,
  RuntimeEventV1Schema,
} from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  buildRuntimeEventSearchParams,
  filterRuntimeEvents,
  mergeRuntimeEvents,
  runtimeEventOccurredAtMillis,
} from './runtimeEventFeed';

interface EventOverrides {
  id?: string;
  message?: string;
  source?: RuntimeEventSource;
  severity?: RuntimeEventSeverity;
  workflowId?: string;
  occurredAtSeconds?: bigint;
  occurredAtNanos?: number;
}

function makeEvent(overrides: EventOverrides = {}): RuntimeEventV1 {
  return create(RuntimeEventV1Schema, {
    id: overrides.id ?? 'event-1',
    message: overrides.message ?? 'event',
    source: overrides.source ?? RuntimeEventSource.WORKER,
    severity: overrides.severity ?? RuntimeEventSeverity.INFO,
    workflowId: overrides.workflowId ?? 'wf-1',
    occurredAt: create(TimestampSchema, {
      seconds: overrides.occurredAtSeconds ?? 10n,
      nanos: overrides.occurredAtNanos ?? 0,
    }),
  });
}

describe('runtimeEventFeed helpers', () => {
  it('sorts merged events from newest to oldest and dedupes by id', () => {
    const older = makeEvent({ id: 'same', message: 'older', occurredAtSeconds: 10n });
    const newerReplacement = makeEvent({ id: 'same', message: 'newer', occurredAtSeconds: 12n });
    const newest = makeEvent({ id: 'two', message: 'newest', occurredAtSeconds: 15n });

    const merged = mergeRuntimeEvents([older], [newerReplacement, newest]);

    expect(merged.map((event) => event.id)).toEqual(['two', 'same']);
    expect(merged[1].message).toBe('newer');
  });

  it('filters by client-side severity and source', () => {
    const events = [
      makeEvent({ id: 'worker-info', source: RuntimeEventSource.WORKER, severity: RuntimeEventSeverity.INFO }),
      makeEvent({ id: 'server-error', source: RuntimeEventSource.SERVER, severity: RuntimeEventSeverity.ERROR }),
    ];

    expect(filterRuntimeEvents(events, { source: RuntimeEventSource.SERVER }).map((event) => event.id)).toEqual([
      'server-error',
    ]);
    expect(filterRuntimeEvents(events, { severity: RuntimeEventSeverity.INFO }).map((event) => event.id)).toEqual([
      'worker-info',
    ]);
  });

  it('builds stable query strings for server filters', () => {
    expect(buildRuntimeEventSearchParams({
      workflowId: 'wf-1',
      opId: 'op-1',
      site: 'example',
      workerId: 'worker-a',
      limit: 25,
    })).toBe('workflowId=wf-1&opId=op-1&site=example&workerId=worker-a&limit=25');
  });

  it('converts protobuf timestamps to epoch milliseconds', () => {
    expect(runtimeEventOccurredAtMillis(makeEvent({
      occurredAtSeconds: 20n,
      occurredAtNanos: 250_000_000,
    }))).toBe(20_250);
  });
});
