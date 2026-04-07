import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type {
  WorkflowListItem,
  WorkflowSummary,
  WorkflowOp,
  OpResult,
  ArtifactSummary,
  WorkflowArtifactListResponse,
  WorkflowResultSummary,
  WorkflowResultsResponse,
} from './types';
import { engineApi } from './engineApi';
import { queueApi } from './queueApi';
import { runtimeEventsApi } from './runtimeEventsApi';

interface ListWorkflowsParams {
  site?: string;
  status?: string;
  limit?: number;
  offset?: number;
}

interface ListWorkflowsResponse {
  workflows: WorkflowListItem[];
  total: number;
}

export const workflowApi = createApi({
  reducerPath: 'workflowApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  tagTypes: ['WorkflowList', 'Workflow', 'WorkflowOps', 'OpResult', 'OpArtifacts', 'WorkflowArtifacts', 'WorkflowResults'],
  endpoints: (builder) => ({
    listWorkflows: builder.query<ListWorkflowsResponse, ListWorkflowsParams>({
      query: (params) => {
        const searchParams = new URLSearchParams();
        if (params.site) searchParams.set('site', params.site);
        if (params.status) searchParams.set('status', params.status);
        if (params.limit) searchParams.set('limit', String(params.limit));
        if (params.offset) searchParams.set('offset', String(params.offset));
        return `/workflows?${searchParams.toString()}`;
      },
      providesTags: (result) => [
        { type: 'WorkflowList', id: 'LIST' },
        ...(result?.workflows?.map(({ workflow }) => ({ type: 'Workflow' as const, id: workflow.ID })) ?? []),
      ],
    }),
    getWorkflow: builder.query<WorkflowSummary, string>({
      query: (id) => `/workflows/${id}`,
      transformResponse: (response: { workflow: WorkflowSummary }) => response.workflow,
      providesTags: (_result, _error, id) => [{ type: 'Workflow', id }],
    }),
    getWorkflowOps: builder.query<WorkflowOp[], string>({
      query: (id) => `/workflows/${id}/ops`,
      transformResponse: (response: { ops: WorkflowOp[] }) => response.ops,
      providesTags: (_result, _error, id) => [{ type: 'WorkflowOps', id }],
    }),
    getOpResult: builder.query<OpResult | null, { workflowId: string; opId: string }>({
      query: ({ workflowId, opId }) => `/workflows/${workflowId}/ops/${opId}/result`,
      transformResponse: (response: { result: OpResult | null }) => response.result,
      providesTags: (_result, _error, { workflowId, opId }) => [{ type: 'OpResult', id: `${workflowId}:${opId}` }],
    }),
    getWorkflowArtifacts: builder.query<WorkflowArtifactListResponse, {
      workflowId: string;
      opId?: string;
      kind?: string;
      contentType?: string;
      search?: string;
      limit?: number;
      offset?: number;
    }>({
      query: ({ workflowId, opId, kind, contentType, search, limit = 20, offset = 0 }) => {
        const sp = new URLSearchParams();
        if (opId) sp.set('opId', opId);
        if (kind) sp.set('kind', kind);
        if (contentType) sp.set('contentType', contentType);
        if (search) sp.set('search', search);
        sp.set('limit', String(limit));
        sp.set('offset', String(offset));
        return `/workflows/${workflowId}/artifacts?${sp}`;
      },
      // Keep the full response so callers can access both artifacts[] and total via
      // selectFromResult. For a simple artifacts-only slice, use:
      //   selectFromResult: (r) => r.data ?? []
      providesTags: (_result, _error, { workflowId }) => [
        { type: 'WorkflowArtifacts', id: workflowId },
      ],
    }),
    getWorkflowResults: builder.query<WorkflowResultsResponse, {
      workflowId: string;
      opId?: string;
      kind?: string;
      status?: string;
      search?: string;
      limit?: number;
      offset?: number;
    }>({
      query: ({ workflowId, opId, kind, status, search, limit = 20, offset = 0 }) => {
        const sp = new URLSearchParams();
        if (opId) sp.set('opId', opId);
        if (kind) sp.set('kind', kind);
        if (status) sp.set('status', status);
        if (search) sp.set('search', search);
        sp.set('limit', String(limit));
        sp.set('offset', String(offset));
        return `/workflows/${workflowId}/results?${sp}`;
      },
      providesTags: (_result, _error, { workflowId }) => [
        { type: 'WorkflowResults', id: workflowId },
      ],
    }),
    getOpArtifacts: builder.query<ArtifactSummary[], { wfId: string; opId: string }>({
      query: ({ wfId, opId }) => `/workflows/${wfId}/ops/${opId}/artifacts`,
      transformResponse: (response: { artifacts: ArtifactSummary[] }) => response.artifacts,
      providesTags: (_result, _error, { wfId, opId }) => [{ type: 'OpArtifacts', id: `${wfId}:${opId}` }],
    }),
    retryOp: builder.mutation<void, { wfId: string; opId: string }>({
      query: ({ wfId, opId }) => ({
        url: `/workflows/${wfId}/ops/${opId}:retry`,
        method: 'POST',
      }),
      invalidatesTags: (_result, _error, { wfId, opId }) => [
        { type: 'WorkflowList', id: 'LIST' },
        { type: 'Workflow', id: wfId },
        { type: 'WorkflowOps', id: wfId },
        { type: 'OpResult', id: `${wfId}:${opId}` },
        { type: 'OpArtifacts', id: `${wfId}:${opId}` },
      ],
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          await queryFulfilled;
          dispatch(engineApi.util.invalidateTags(['EngineStatus']));
          dispatch(queueApi.util.invalidateTags(['QueueStatus']));
          dispatch(runtimeEventsApi.util.invalidateTags(['RuntimeEvents']));
        } catch {
          // no cache invalidation on failed retry
        }
      },
    }),
    cancelWorkflow: builder.mutation<void, string>({
      query: (wfId) => ({
        url: `/workflows/${wfId}:cancel`,
        method: 'POST',
      }),
      invalidatesTags: (_result, _error, wfId) => [
        { type: 'WorkflowList', id: 'LIST' },
        { type: 'Workflow', id: wfId },
        { type: 'WorkflowOps', id: wfId },
      ],
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          await queryFulfilled;
          dispatch(engineApi.util.invalidateTags(['EngineStatus']));
          dispatch(queueApi.util.invalidateTags(['QueueStatus']));
          dispatch(runtimeEventsApi.util.invalidateTags(['RuntimeEvents']));
        } catch {
          // no cache invalidation on failed cancellation
        }
      },
    }),
  }),
});

export const {
  useListWorkflowsQuery,
  useGetWorkflowQuery,
  useGetWorkflowOpsQuery,
  useGetOpResultQuery,
  useGetWorkflowArtifactsQuery,
  useGetWorkflowResultsQuery,
  useGetOpArtifactsQuery,
  useRetryOpMutation,
  useCancelWorkflowMutation,
} = workflowApi;
