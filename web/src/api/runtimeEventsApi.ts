import { fromJson } from '@bufbuild/protobuf';
import type { JsonValue } from '@bufbuild/protobuf';
import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import { RuntimeEventV1Schema, type RuntimeEventV1 } from '../pb/proto/scraper/runtime/v1/events_pb';

interface RuntimeEventsResponse {
  events: JsonValue[];
}

interface RuntimeEventsParams {
  workflowId?: string;
  opId?: string;
  site?: string;
  workerId?: string;
  limit?: number;
}

export function decodeRuntimeEvent(json: JsonValue): RuntimeEventV1 {
  return fromJson(RuntimeEventV1Schema, json);
}

export const runtimeEventsApi = createApi({
  reducerPath: 'runtimeEventsApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['RuntimeEvents'],
  endpoints: (builder) => ({
    getRecentRuntimeEvents: builder.query<RuntimeEventV1[], RuntimeEventsParams>({
      query: (params) => {
        const searchParams = new URLSearchParams();
        if (params.workflowId) searchParams.set('workflowId', params.workflowId);
        if (params.opId) searchParams.set('opId', params.opId);
        if (params.site) searchParams.set('site', params.site);
        if (params.workerId) searchParams.set('workerId', params.workerId);
        if (params.limit) searchParams.set('limit', String(params.limit));
        return `/runtime-events?${searchParams.toString()}`;
      },
      transformResponse: (response: RuntimeEventsResponse) =>
        response.events.map((event) => decodeRuntimeEvent(event)),
      providesTags: ['RuntimeEvents'],
    }),
  }),
});

export const { useGetRecentRuntimeEventsQuery } = runtimeEventsApi;
