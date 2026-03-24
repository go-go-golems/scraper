import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { QueueStatus } from './types';

export const queueApi = createApi({
  reducerPath: 'queueApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['QueueStatus'],
  endpoints: (builder) => ({
    listQueues: builder.query<QueueStatus[], void>({
      query: () => '/queues',
      transformResponse: (response: { queues: QueueStatus[] }) => response.queues,
      providesTags: ['QueueStatus'],
    }),
  }),
});

export const { useListQueuesQuery } = queueApi;
