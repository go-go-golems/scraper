import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { WorkflowListItem, WorkflowSummary, WorkflowOp, OpResult, ArtifactSummary } from './types';

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
  tagTypes: ['WorkflowList', 'Workflow', 'WorkflowOps'],
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
      providesTags: ['WorkflowList'],
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
    }),
    getOpArtifacts: builder.query<ArtifactSummary[], { wfId: string; opId: string }>({
      query: ({ wfId, opId }) => `/workflows/${wfId}/ops/${opId}/artifacts`,
      transformResponse: (response: { artifacts: ArtifactSummary[] }) => response.artifacts,
    }),
    retryOp: builder.mutation<void, { wfId: string; opId: string }>({
      query: ({ wfId, opId }) => ({
        url: `/workflows/${wfId}/ops/${opId}:retry`,
        method: 'POST',
      }),
      invalidatesTags: ['WorkflowOps'],
    }),
    cancelWorkflow: builder.mutation<void, string>({
      query: (wfId) => ({
        url: `/workflows/${wfId}:cancel`,
        method: 'POST',
      }),
      invalidatesTags: ['Workflow', 'WorkflowList'],
    }),
  }),
});

export const {
  useListWorkflowsQuery,
  useGetWorkflowQuery,
  useGetWorkflowOpsQuery,
  useGetOpResultQuery,
  useGetOpArtifactsQuery,
  useRetryOpMutation,
  useCancelWorkflowMutation,
} = workflowApi;
