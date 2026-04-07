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
  filterRuntimeEvents,
  mergeRuntimeEvents,
} from './runtimeEventHelpers';
import {
  runtimeEventOccurredAtMillis,
  type RuntimeEventsParams,
} from '../../api/runtimeEventsApi';

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

function buildSearchParams(params: RuntimeEventsParams): string {
  const searchParams = new URLSearchParams();
  if (params.workflowId) searchParams.set('workflowId', params.workflowId);
  if (params.opId) searchParams.set('opId', params.opId);
  if (params.site) searchParams.set('site', params.site);
  if (params.workerId) searchParams.set('workerId', params.workerId);
  if (params.limit) searchParams.set('limit', String(params.limit));
  if (params.since) searchParams.set('since', params.since);
  if (params.until) searchParams.set('until', params.until);
  if (params.offset) searchParams.set('offset', String(params.offset));
  return searchParams.toString();
}

describe('runtimeEvent helpers', () => {
  it('sorts merged events from newest to oldest and dedupes by id', () => {
    const older = makeEvent({ id: 'same', message: 'older', occurredAtSeconds: 10n });
    const newerReplacement = makeEvent({ id: 'same', message: 'newer', occurredAtSeconds: 12n });
    const newest = makeEvent({ id: 'two', message: 'newest', occurredAtSeconds: 15n });

    const merged = mergeRuntimeEvents([older], [newerReplacement, newest], runtimeEventOccurredAtMillis);

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
    expect(buildSearchParams({
      workflowId: 'wf-1',
      opId: 'op-1',
      site: 'example',
      workerId: 'worker_a',
      limit: 25,
    })).toBe('workflowId=wf-1&opId=op-1&site=example&workerId=worker_a&limit=25');
  });

  it('converts protobuf timestamps to epoch milliseconds', () => {
    expect(runtimeEventOccurredAtMillis(makeEvent({
      occurredAtSeconds: 20n,
      occurredAtNanos: 250_000_000,
    }))).toBe(20_250);
  });
});
