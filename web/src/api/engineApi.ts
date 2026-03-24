import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { EngineStatus } from './types';

export const engineApi = createApi({
  reducerPath: 'engineApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['EngineStatus'],
  endpoints: (builder) => ({
    getEngineStatus: builder.query<EngineStatus, void>({
      query: () => '/engine/status',
      transformResponse: (response: { status: EngineStatus }) => response.status,
      providesTags: ['EngineStatus'],
    }),
  }),
});

export const { useGetEngineStatusQuery } = engineApi;
