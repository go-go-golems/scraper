import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { SiteSummary, VerbSummary } from './types';

export const catalogApi = createApi({
  reducerPath: 'catalogApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['Sites', 'Verbs'],
  endpoints: (builder) => ({
    listSites: builder.query<SiteSummary[], void>({
      query: () => '/sites',
      transformResponse: (response: { sites: SiteSummary[] }) => response.sites,
      providesTags: ['Sites'],
    }),
    listVerbs: builder.query<VerbSummary[], string>({
      query: (site) => `/sites/${site}/verbs`,
      transformResponse: (response: { verbs: VerbSummary[] }) => response.verbs,
      providesTags: (_result, _error, site) => [{ type: 'Verbs', id: site }],
    }),
  }),
});

export const { useListSitesQuery, useListVerbsQuery } = catalogApi;
