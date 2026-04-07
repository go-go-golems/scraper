import { fromJson } from '@bufbuild/protobuf';
import type { JsonValue } from '@bufbuild/protobuf';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { RuntimeEventV1Schema, type RuntimeEventV1 } from '../pb/proto/scraper/runtime/v1/events_pb';

// ── types ────────────────────────────────────────────────────────────

interface RuntimeEventsResponse {
  events: JsonValue[];
}

export interface RuntimeEventsParams {
  workflowId?: string;
  opId?: string;
  site?: string;
  workerId?: string;
  limit?: number;
  since?: string;
  until?: string;
  offset?: number;
}

export type RuntimeEventJson = JsonValue;

// ── helpers ──────────────────────────────────────────────────────────

export function decodeRuntimeEvent(json: JsonValue): RuntimeEventV1 {
  return fromJson(RuntimeEventV1Schema, json);
}

export function runtimeEventOccurredAtMillis(event: RuntimeEventV1): number {
  return (
    Number(event.occurredAt?.seconds ?? 0n) * 1000 +
    Math.floor((event.occurredAt?.nanos ?? 0) / 1_000_000)
  );
}

function buildRuntimeEventQuery(params: RuntimeEventsParams): string {
  const searchParams = new URLSearchParams();
  if (params.workflowId) searchParams.set('workflowId', params.workflowId);
  if (params.opId) searchParams.set('opId', params.opId);
  if (params.site) searchParams.set('site', params.site);
  if (params.workerId) searchParams.set('workerId', params.workerId);
  if (params.limit) searchParams.set('limit', String(params.limit));
  if (params.since) searchParams.set('since', params.since);
  if (params.until) searchParams.set('until', params.until);
  if (params.offset) searchParams.set('offset', String(params.offset));
  return `/runtime-events?${searchParams.toString()}`;
}

function buildSSEUrl(params: RuntimeEventsParams): string {
  const searchParams = new URLSearchParams();
  if (params.workflowId) searchParams.set('workflowId', params.workflowId);
  if (params.opId) searchParams.set('opId', params.opId);
  if (params.site) searchParams.set('site', params.site);
  if (params.workerId) searchParams.set('workerId', params.workerId);
  const search = searchParams.toString();
  return search
    ? `/api/v1/runtime-events/stream?${search}`
    : '/api/v1/runtime-events/stream';
}

const MAX_CACHED_EVENTS = 500;

// ── API ──────────────────────────────────────────────────────────────

export const runtimeEventsApi = createApi({
  reducerPath: 'runtimeEventsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['RuntimeEvents'],
  endpoints: (builder) => ({
    getRecentRuntimeEvents: builder.query<RuntimeEventV1[], RuntimeEventsParams>({
      query: (params) => buildRuntimeEventQuery(params),
      transformResponse: (response: RuntimeEventsResponse) =>
        response.events.map((json) => decodeRuntimeEvent(json)),
      providesTags: ['RuntimeEvents'],
      keepUnusedDataFor: 30,

      async onCacheEntryAdded(
        arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved },
      ) {
        // Wait for the initial REST fetch to resolve.
        // If there is no backend, cacheDataLoaded rejects and we return early —
        // no SSE connection is opened, no infinite loop.
        try {
          await cacheDataLoaded;
        } catch {
          return;
        }

        // Open SSE stream with the same scoping filters
        const sseUrl = buildSSEUrl(arg);
        const eventSource = new EventSource(sseUrl);

        const onMessage = (event: MessageEvent<string>) => {
          try {
            const decoded = decodeRuntimeEvent(JSON.parse(event.data));
            updateCachedData((draft) => {
              // Dedupe
              const exists = draft.some((e) => e.id === decoded.id);
              if (!exists) draft.unshift(decoded);
              // Sort newest-first
              draft.sort(
                (a, b) =>
                  runtimeEventOccurredAtMillis(b) - runtimeEventOccurredAtMillis(a),
              );
              // Trim to max
              if (draft.length > MAX_CACHED_EVENTS) {
                draft.length = MAX_CACHED_EVENTS;
              }
            });
          } catch {
            // ignore malformed event payloads
          }
        };

        eventSource.addEventListener('runtime-event', onMessage as EventListener);

        // Close SSE on error to prevent browser auto-reconnect
        eventSource.addEventListener('error', () => {
          eventSource.close();
        });

        // Auto-cleanup when no subscribers remain
        await cacheEntryRemoved;
        eventSource.close();
      },
    }),
  }),
});

export const { useGetRecentRuntimeEventsQuery } = runtimeEventsApi;
