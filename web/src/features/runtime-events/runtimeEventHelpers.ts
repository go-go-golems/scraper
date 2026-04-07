/**
 * Pure helper functions for runtime events.
 * Extracted from the old useRuntimeEventFeed hook so they can be
 * shared between the RTK Query API layer and UI consumers.
 */

import type { RuntimeEventV1 } from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../../pb/proto/scraper/runtime/v1/events_pb';

// ── re-export from API for convenience ───────────────────────────────

export {
  decodeRuntimeEvent,
  runtimeEventOccurredAtMillis,
} from '../../api/runtimeEventsApi';

export type {
  RuntimeEventsParams,
  RuntimeEventJson,
} from '../../api/runtimeEventsApi';

// ── client-side filtering ────────────────────────────────────────────

export interface RuntimeEventClientFilters {
  severity?: RuntimeEventSeverity | 'all';
  source?: RuntimeEventSource | 'all';
}

export function filterRuntimeEvents(
  events: RuntimeEventV1[],
  clientFilters: RuntimeEventClientFilters = {},
): RuntimeEventV1[] {
  return events.filter((event) => {
    if (
      clientFilters.severity &&
      clientFilters.severity !== 'all' &&
      event.severity !== clientFilters.severity
    ) {
      return false;
    }

    if (
      clientFilters.source &&
      clientFilters.source !== 'all' &&
      event.source !== clientFilters.source
    ) {
      return false;
    }

    return true;
  });
}

// ── merge utility (still useful for tests and edge cases) ────────────

export function mergeRuntimeEvents(
  current: RuntimeEventV1[],
  incoming: RuntimeEventV1[],
  sortFn: (e: RuntimeEventV1) => number,
): RuntimeEventV1[] {
  const byId = new Map<string, RuntimeEventV1>();

  for (const event of current) {
    byId.set(event.id, event);
  }

  for (const event of incoming) {
    byId.set(event.id, event);
  }

  return Array.from(byId.values()).sort(
    (left, right) => sortFn(right) - sortFn(left),
  );
}
