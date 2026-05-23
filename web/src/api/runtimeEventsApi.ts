import { fromJson, toJson } from '@bufbuild/protobuf';
import type { JsonObject, JsonValue } from '@bufbuild/protobuf';
import { createApi, fakeBaseQuery } from '@reduxjs/toolkit/query/react';
import {
  RuntimeEventAppendedSchema,
  RuntimeEventEntitySchema,
} from '../pb/proto/scraper/runtime/sessionstream/v1/runtime_stream_pb';
import { RuntimeEventV1Schema, type RuntimeEventV1 } from '../pb/proto/scraper/runtime/v1/events_pb';

// ── types ────────────────────────────────────────────────────────────

export interface RuntimeEventsParams {
  workflowId?: string;
  opId?: string;
  site?: string;
  workerId?: string;
  severities?: string[];
  sources?: string[];
  limit?: number;
  since?: string;
  until?: string;
  offset?: number;
}

export type RuntimeEventJson = JsonValue;

type ServerFrameJson = JsonObject & {
  hello?: JsonObject;
  snapshot?: {
    sessionId?: string;
    snapshotOrdinal?: string | number | bigint;
    entities?: SnapshotEntityJson[];
  };
  subscribed?: JsonObject;
  uiEvent?: {
    sessionId?: string;
    eventOrdinal?: string | number | bigint;
    name?: string;
    payload?: JsonValue;
  };
  error?: JsonObject;
};

interface SnapshotEntityJson {
  kind?: string;
  id?: string;
  payload?: JsonValue;
  tombstone?: boolean;
}

const UI_EVENT_RUNTIME_EVENT_APPENDED = 'scraper.runtime.RuntimeEventAppended';
const ENTITY_RUNTIME_EVENT = 'scraper.runtime.RuntimeEvent';
const MAX_CACHED_EVENTS = 500;

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

function runtimeEventToJson(event: RuntimeEventV1): RuntimeEventJson {
  return toJson(RuntimeEventV1Schema, event);
}

function runtimeEventSession(params: RuntimeEventsParams): string {
  if (params.workflowId) return `workflow:${params.workflowId}`;
  return 'runtime:global';
}

function runtimeEventsWebSocketUrl(): string {
  if (typeof window === 'undefined') return '/api/v1/runtime-events/ws';
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}/api/v1/runtime-events/ws`;
}

function subscribeFrame(sessionId: string): string {
  return JSON.stringify({ subscribe: { sessionId, sinceSnapshotOrdinal: '0' } });
}

function stripAnyType(payload: JsonValue | undefined): JsonValue | undefined {
  if (!payload || typeof payload !== 'object' || Array.isArray(payload)) return payload;
  const { ['@type']: _type, ...rest } = payload as JsonObject;
  return rest as JsonObject;
}

function runtimeEventFromSnapshotEntity(entity: SnapshotEntityJson): RuntimeEventV1 | undefined {
  if (entity.kind !== ENTITY_RUNTIME_EVENT || entity.tombstone || !entity.payload) return undefined;
  const decoded = fromJson(RuntimeEventEntitySchema, stripAnyType(entity.payload) as JsonValue);
  return decoded.event;
}

function runtimeEventFromUIEvent(frame: ServerFrameJson['uiEvent']): RuntimeEventV1 | undefined {
  if (!frame || frame.name !== UI_EVENT_RUNTIME_EVENT_APPENDED || !frame.payload) return undefined;
  const decoded = fromJson(RuntimeEventAppendedSchema, stripAnyType(frame.payload) as JsonValue);
  return decoded.event;
}

function matchesParams(event: RuntimeEventV1, params: RuntimeEventsParams): boolean {
  if (params.workflowId && event.workflowId !== params.workflowId) return false;
  if (params.opId && event.opId !== params.opId) return false;
  if (params.site && event.site !== params.site) return false;
  if (params.workerId && event.workerId !== params.workerId) return false;
  if (params.severities?.length && !params.severities.includes(String(event.severity))) return false;
  if (params.sources?.length && !params.sources.includes(String(event.source))) return false;
  const occurredAt = runtimeEventOccurredAtMillis(event);
  if (params.since && occurredAt < Date.parse(params.since)) return false;
  if (params.until && occurredAt > Date.parse(params.until)) return false;
  return true;
}

function mergeRuntimeEventJson(current: RuntimeEventJson[], incoming: RuntimeEventV1[], params: RuntimeEventsParams): RuntimeEventJson[] {
  const byId = new Map<string, RuntimeEventV1>();
  for (const raw of current) {
    try {
      const event = decodeRuntimeEvent(raw);
      if (event.id) byId.set(event.id, event);
    } catch {
      // ignore malformed cached entries
    }
  }
  for (const event of incoming) {
    if (!matchesParams(event, params)) continue;
    if (event.id) byId.set(event.id, event);
  }
  const limit = params.limit && params.limit > 0 ? params.limit : MAX_CACHED_EVENTS;
  return [...byId.values()]
    .sort((a, b) => runtimeEventOccurredAtMillis(b) - runtimeEventOccurredAtMillis(a))
    .slice(0, limit)
    .map(runtimeEventToJson);
}

function isStorybookEnvironment(): boolean {
  return (
    typeof window !== 'undefined' &&
    '__STORYBOOK_PREVIEW__' in window
  );
}

// ── API ──────────────────────────────────────────────────────────────

export const runtimeEventsApi = createApi({
  reducerPath: 'runtimeEventsApi',
  baseQuery: fakeBaseQuery(),
  tagTypes: ['RuntimeEvents'],
  endpoints: (builder) => ({
    getRecentRuntimeEvents: builder.query<RuntimeEventJson[], RuntimeEventsParams>({
      queryFn: () => ({ data: [] }),
      providesTags: ['RuntimeEvents'],
      keepUnusedDataFor: 30,

      async onCacheEntryAdded(
        arg,
        { updateCachedData, cacheDataLoaded, cacheEntryRemoved },
      ) {
        await cacheDataLoaded;

        if (typeof window === 'undefined' || isStorybookEnvironment()) {
          await cacheEntryRemoved;
          return;
        }

        const socket = new WebSocket(runtimeEventsWebSocketUrl());
        const sessionId = runtimeEventSession(arg);

        socket.addEventListener('open', () => {
          socket.send(subscribeFrame(sessionId));
        });

        socket.addEventListener('message', (message: MessageEvent<string>) => {
          try {
            const frame = JSON.parse(message.data) as ServerFrameJson;
            if (frame.snapshot?.entities) {
              const events = frame.snapshot.entities
                .map(runtimeEventFromSnapshotEntity)
                .filter((event): event is RuntimeEventV1 => Boolean(event));
              updateCachedData((draft) => {
                const merged = mergeRuntimeEventJson([], events, arg);
                draft.splice(0, draft.length, ...merged);
              });
              return;
            }
            const event = runtimeEventFromUIEvent(frame.uiEvent);
            if (!event) return;
            updateCachedData((draft) => {
              const merged = mergeRuntimeEventJson(draft as RuntimeEventJson[], [event], arg);
              draft.splice(0, draft.length, ...merged);
            });
          } catch {
            // Ignore malformed websocket frames. The websocket close/error path
            // below controls lifecycle; individual bad frames should not tear
            // down the RTK Query cache entry.
          }
        });

        await cacheEntryRemoved;
        socket.close();
      },
    }),
  }),
});

export const { useGetRecentRuntimeEventsQuery } = runtimeEventsApi;
