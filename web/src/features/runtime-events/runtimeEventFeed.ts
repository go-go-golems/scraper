import { useEffect, useState } from 'react';
import type { RuntimeEventV1 } from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  decodeRuntimeEvent,
  useGetRecentRuntimeEventsQuery,
  type RuntimeEventJson,
  type RuntimeEventsParams,
} from '../../api/runtimeEventsApi';

export type RuntimeEventConnectionState = 'connecting' | 'live' | 'error' | 'closed';

export interface RuntimeEventClientFilters {
  severity?: RuntimeEventSeverity | 'all';
  source?: RuntimeEventSource | 'all';
}

export interface UseRuntimeEventFeedOptions {
  serverFilters: RuntimeEventsParams;
  clientFilters?: RuntimeEventClientFilters;
  stream?: boolean;
}

export function runtimeEventOccurredAtMillis(event: RuntimeEventV1): number {
  return Number(event.occurredAt?.seconds ?? 0n) * 1000 + Math.floor((event.occurredAt?.nanos ?? 0) / 1_000_000);
}

export function mergeRuntimeEvents(current: RuntimeEventV1[], incoming: RuntimeEventV1[]): RuntimeEventV1[] {
  const byId = new Map<string, RuntimeEventV1>();

  for (const event of current) {
    byId.set(event.id, event);
  }

  for (const event of incoming) {
    byId.set(event.id, event);
  }

  return Array.from(byId.values()).sort((left, right) => runtimeEventOccurredAtMillis(right) - runtimeEventOccurredAtMillis(left));
}

function decodeRuntimeEvents(events: RuntimeEventJson[]): RuntimeEventV1[] {
  return events.map((event) => decodeRuntimeEvent(event));
}

export function filterRuntimeEvents(
  events: RuntimeEventV1[],
  clientFilters: RuntimeEventClientFilters = {},
): RuntimeEventV1[] {
  return events.filter((event) => {
    if (clientFilters.severity && clientFilters.severity !== 'all' && event.severity !== clientFilters.severity) {
      return false;
    }

    if (clientFilters.source && clientFilters.source !== 'all' && event.source !== clientFilters.source) {
      return false;
    }

    return true;
  });
}

export function buildRuntimeEventSearchParams(filters: RuntimeEventsParams): string {
  const searchParams = new URLSearchParams();

  if (filters.workflowId) searchParams.set('workflowId', filters.workflowId);
  if (filters.opId) searchParams.set('opId', filters.opId);
  if (filters.site) searchParams.set('site', filters.site);
  if (filters.workerId) searchParams.set('workerId', filters.workerId);
  if (filters.limit) searchParams.set('limit', String(filters.limit));

  return searchParams.toString();
}

export function useRuntimeEventFeed({
  serverFilters,
  clientFilters,
  stream = true,
}: UseRuntimeEventFeedOptions) {
  const [allEvents, setAllEvents] = useState<RuntimeEventV1[]>([]);
  const [connectionState, setConnectionState] = useState<RuntimeEventConnectionState>(stream ? 'connecting' : 'closed');
  const [lastEventAt, setLastEventAt] = useState<number | null>(null);
  const search = buildRuntimeEventSearchParams(serverFilters);

  const { data: recentRuntimeEvents = [], isLoading } = useGetRecentRuntimeEventsQuery(serverFilters);

  useEffect(() => {
    setAllEvents([]);
    setLastEventAt(null);
    setConnectionState(stream ? 'connecting' : 'closed');
  }, [search, stream]);

  useEffect(() => {
    setAllEvents((current) => mergeRuntimeEvents(current, decodeRuntimeEvents(recentRuntimeEvents)));
  }, [recentRuntimeEvents]);

  useEffect(() => {
    if (allEvents.length === 0) {
      setLastEventAt(null);
      return;
    }

    setLastEventAt(runtimeEventOccurredAtMillis(allEvents[0]));
  }, [allEvents]);

  useEffect(() => {
    if (!stream) {
      setConnectionState('closed');
      return undefined;
    }

    setConnectionState('connecting');

    const streamUrl = search.length > 0 ? `/api/v1/runtime-events/stream?${search}` : '/api/v1/runtime-events/stream';
    const eventSource = new EventSource(streamUrl);

    const onOpen = () => {
      setConnectionState('live');
    };

    const onMessage = (event: MessageEvent<string>) => {
      try {
        const decoded = decodeRuntimeEvent(JSON.parse(event.data));
        setAllEvents((current) => mergeRuntimeEvents(current, [decoded]));
        setConnectionState('live');
      } catch {
        // ignore malformed event payloads
      }
    };

    const onError = () => {
      setConnectionState('error');
    };

    eventSource.addEventListener('open', onOpen as EventListener);
    eventSource.addEventListener('runtime-event', onMessage as EventListener);
    eventSource.addEventListener('error', onError as EventListener);

    return () => {
      eventSource.removeEventListener('open', onOpen as EventListener);
      eventSource.removeEventListener('runtime-event', onMessage as EventListener);
      eventSource.removeEventListener('error', onError as EventListener);
      eventSource.close();
      setConnectionState('closed');
    };
  }, [search, stream]);

  return {
    allEvents,
    events: filterRuntimeEvents(allEvents, clientFilters),
    isLoadingHistory: isLoading,
    connectionState,
    lastEventAt,
  };
}
