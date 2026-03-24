import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { WorkflowRun } from './types';

interface SubmitRequest {
  site: string;
  verb: string;
  workflowId?: string;
  values: Record<string, unknown>;
}

interface SubmitResponse {
  site: string;
  verb: string;
  workflow: WorkflowRun;
  targetOpID: string;
  submittedCount: number;
}

export const submissionApi = createApi({
  reducerPath: 'submissionApi',
  baseQuery: fetchBaseQuery({ baseUrl: '/api/v1' }),
  endpoints: (builder) => ({
    submitWorkflow: builder.mutation<SubmitResponse, SubmitRequest>({
      query: ({ site, verb, workflowId, values }) => ({
        url: `/sites/${site}/verbs/${verb}:submit`,
        method: 'POST',
        body: { workflowID: workflowId, values },
      }),
    }),
  }),
});

export const { useSubmitWorkflowMutation } = submissionApi;
