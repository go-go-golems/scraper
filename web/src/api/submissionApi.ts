import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';
import type { WorkflowRun } from './types';
import { engineApi } from './engineApi';
import { queueApi } from './queueApi';
import { runtimeEventsApi } from './runtimeEventsApi';
import { workflowApi } from './workflowApi';

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
      async onQueryStarted(_arg, { dispatch, queryFulfilled }) {
        try {
          const { data } = await queryFulfilled;
          const submittedWorkflowId = data.workflow.ID;

          dispatch(engineApi.util.invalidateTags(['EngineStatus']));
          dispatch(queueApi.util.invalidateTags(['QueueStatus']));
          dispatch(runtimeEventsApi.util.invalidateTags(['RuntimeEvents']));
          dispatch(
            workflowApi.util.invalidateTags([
              { type: 'WorkflowList', id: 'LIST' },
              { type: 'Workflow', id: submittedWorkflowId },
              { type: 'WorkflowOps', id: submittedWorkflowId },
            ]),
          );
        } catch {
          // no cache invalidation on failed submission
        }
      },
    }),
  }),
});

export const { useSubmitWorkflowMutation } = submissionApi;
