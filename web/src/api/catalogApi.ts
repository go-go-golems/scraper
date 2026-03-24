import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { SiteDetail, SiteSummary, VerbSummary } from './types';

export interface ScriptDetail {
  path: string;
  source: string;
}

export const catalogApi = createApi({
  reducerPath: 'catalogApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['Sites', 'Verbs', 'Scripts'],
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
    listScripts: builder.query<string[], string>({
      query: (site) => `/sites/${site}/scripts`,
      transformResponse: (response: { scripts: string[] }) => response.scripts,
      providesTags: (_result, _error, site) => [{ type: 'Scripts', id: site }],
    }),
    getSiteDetail: builder.query<SiteDetail, string>({
      query: (site) => `/sites/${site}/detail`,
      transformResponse: (response: { site: SiteDetail }) => response.site,
      providesTags: (_result, _error, site) => [{ type: 'Sites', id: site }],
    }),
    getScript: builder.query<ScriptDetail, { site: string; path: string }>({
      query: ({ site, path }) => `/sites/${site}/scripts/${path}`,
      transformResponse: (response: ScriptDetail) => response,
      providesTags: (_result, _error, { site, path }) => [
        { type: 'Scripts', id: `${site}:${path}` },
      ],
    }),
  }),
});

export const {
  useListSitesQuery,
  useListVerbsQuery,
  useListScriptsQuery,
  useGetSiteDetailQuery,
  useGetScriptQuery,
} = catalogApi;
