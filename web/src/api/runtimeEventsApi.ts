import { fromJson } from '@bufbuild/protobuf';
import type { JsonValue } from '@bufbuild/protobuf';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { RuntimeEventV1Schema, type RuntimeEventV1 } from '../pb/proto/scraper/runtime/v1/events_pb';

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

export function decodeRuntimeEvent(json: JsonValue): RuntimeEventV1 {
  return fromJson(RuntimeEventV1Schema, json);
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

export const runtimeEventsApi = createApi({
  reducerPath: 'runtimeEventsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['RuntimeEvents'],
  endpoints: (builder) => ({
    getRecentRuntimeEvents: builder.query<RuntimeEventJson[], RuntimeEventsParams>({
      query: (params) => buildRuntimeEventQuery(params),
      transformResponse: (response: RuntimeEventsResponse) => response.events,
      providesTags: ['RuntimeEvents'],
    }),
  }),
});

export const { useGetRecentRuntimeEventsQuery } = runtimeEventsApi;
