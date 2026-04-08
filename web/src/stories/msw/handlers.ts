import { http, HttpResponse } from 'msw';
import type { ArtifactSummary } from '../api/types';
import type { WorkflowOp } from '../api/types';
import type { WorkflowResultSummary } from '../api/types';

// ─── Shared fixture data ─────────────────────────────────────────────────────

export const STORY_WORKFLOW_ID = 'story-workflow-000';

export function makeArtifact(overrides: Partial<ArtifactSummary> = {}): ArtifactSummary {
  return {
    id: 'story-art-1',
    opID: `${STORY_WORKFLOW_ID}:extract`,
    workflowID: STORY_WORKFLOW_ID,
    name: 'summary.json',
    kind: 'json-output',
    contentType: 'application/json',
    size: 2048,
    createdAt: new Date(Date.now() - 3600_000).toISOString(),
    previewable: true,
    previewKind: 'json',
    ...overrides,
  };
}

export const STORY_ARTIFACTS: ArtifactSummary[] = [
  makeArtifact({
    id: 'story-art-html',
    name: 'index.html',
    kind: 'http-response-body',
    contentType: 'text/html',
    size: 48_320,
    previewable: true,
    previewKind: 'html',
  }),
  makeArtifact({
    id: 'story-art-json',
    name: 'summary.json',
    kind: 'json-output',
    contentType: 'application/json',
    size: 2_048,
  }),
  makeArtifact({
    id: 'story-art-log',
    name: 'debug.log',
    kind: 'exec-log',
    contentType: 'text/plain',
    size: 12_800,
    previewable: true,
    previewKind: 'text',
  }),
  makeArtifact({
    id: 'story-art-img',
    name: 'screenshot.png',
    kind: 'screenshot',
    contentType: 'image/png',
    size: 1_048_576,
    previewable: false,
  }),
];

export function makeWorkflowOp(overrides: Partial<import('../api/types').WorkflowOp['op']> = {}): WorkflowOp['op'] {
  return {
    ID: `${STORY_WORKFLOW_ID}:extract`,
    WorkflowID: STORY_WORKFLOW_ID,
    Site: 'story-site',
    Kind: 'js',
    Queue: 'q',
    DedupKey: 'k',
    Input: {},
    DependsOn: [],
    Retry: { MaxAttempts: 3, BackoffKind: 'exp', InitialBackoff: 1, MaxBackoff: 60, Multiplier: 2 },
    RetryState: { Attempt: 1, LastError: '' },
    Metadata: {},
    ...overrides,
  };
}

// ─── Default handler set for artifact browser stories ─────────────────────────

export const defaultArtifactHandlers = [
  // Workflow artifact list
  http.get(`/api/v1/workflows/:workflowId/artifacts`, ({ params }) => {
    const { workflowId } = params;
    return HttpResponse.json({
      workflowID: String(workflowId),
      total: STORY_ARTIFACTS.length,
      artifacts: STORY_ARTIFACTS,
    });
  }),

  // Workflow ops (needed by FilterBar's op dropdown)
  http.get(`/api/v1/workflows/:workflowId/ops`, ({ params }) => {
    const { workflowId } = params;
    return HttpResponse.json({
      ops: [
        {
          op: makeWorkflowOp({ ID: `${workflowId}:fetch`, Kind: 'http' }),
          status: 'succeeded',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
        {
          op: makeWorkflowOp({ ID: `${workflowId}:extract`, Kind: 'js' }),
          status: 'succeeded',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
        {
          op: makeWorkflowOp({ ID: `${workflowId}:page-2-fetch`, Kind: 'http' }),
          status: 'succeeded',
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      ],
    });
  }),

  // Artifact body — JSON
  http.get('/api/v1/artifacts/story-art-json', () => {
    return HttpResponse.json({ stories: 30, nextPage: '/news?p=2', scrapedAt: new Date().toISOString() });
  }),

  // Artifact body — HTML
  http.get('/api/v1/artifacts/story-art-html', () => {
    return HttpResponse.text(
      `<!DOCTYPE html>\n<html lang="en">\n<head><title>Hacker News</title></head>\n<body>\n<table class="itemlist">\n<tr class="athing" id="12345">\n<td class="title"><a href="https://example.com">Show HN: A cool project</a></td>\n</tr>\n</table>\n</body>\n</html>`,
      { headers: { 'Content-Type': 'text/html' } },
    );
  }),

  // Artifact body — text/log
  http.get('/api/v1/artifacts/story-art-log', () => {
    return HttpResponse.text(
      `[2026-04-07T14:31:58Z] Starting JS extraction\n[2026-04-07T14:31:59Z] Loaded seed.js script\n[2026-04-07T14:32:00Z] Found 30 items on page 1\n[2026-04-07T14:32:05Z] Extraction complete`,
      { headers: { 'Content-Type': 'text/plain' } },
    );
  }),
];

// ─── Empty artifact list handler ─────────────────────────────────────────────

export const emptyArtifactHandlers = [
  // Empty workflow artifact list
  http.get('/api/v1/workflows/:workflowId/artifacts', ({ params }) => {
    return HttpResponse.json({
      workflowID: String(params.workflowId),
      total: 0,
      artifacts: [],
    });
  }),


  // Ops list (needed by FilterBar's op dropdown)
  http.get('/api/v1/workflows/:workflowId/ops', ({ params }) => {
    return HttpResponse.json({ ops: [] });
  }),
];

// ─── Shared fixture data for results ─────────────────────────────────────────

export const STORY_RESULTS: WorkflowResultSummary[] = [
  {
    opID: `${STORY_WORKFLOW_ID}:fetch`,
    kind: 'http',
    status: 'succeeded',
    recordCount: 0,
    artifactCount: 1,
    dataSize: 48_320,
    error: undefined,
    completedAt: new Date(Date.now() - 7200_000).toISOString(),
  },
  {
    opID: `${STORY_WORKFLOW_ID}:extract`,
    kind: 'js',
    status: 'succeeded',
    recordCount: 30,
    artifactCount: 0,
    dataSize: 2048,
    error: undefined,
    completedAt: new Date(Date.now() - 3600_000).toISOString(),
  },
  {
    opID: `${STORY_WORKFLOW_ID}:page-2-fetch`,
    kind: 'http',
    status: 'succeeded',
    recordCount: 0,
    artifactCount: 1,
    dataSize: 52_800,
    error: undefined,
    completedAt: new Date(Date.now() - 1800_000).toISOString(),
  },
  {
    opID: `${STORY_WORKFLOW_ID}:page-2-extract`,
    kind: 'js',
    status: 'failed',
    recordCount: 0,
    artifactCount: 0,
    dataSize: 0,
    error: {
      Code: 'JSError',
      Message: 'SyntaxError: Unexpected token at line 12',
      Retryable: false,
    },
    completedAt: new Date(Date.now() - 900_000).toISOString(),
  },
];

// ─── Default handler set for results stories ─────────────────────────────────

export const defaultResultsHandlers = [
  // Workflow results list
  http.get('/api/v1/workflows/:workflowId/results', ({ params }) => {
    return HttpResponse.json({
      workflowID: String(params.workflowId),
      total: STORY_RESULTS.length,
      results: STORY_RESULTS,
    });
  }),

  // Workflow ops (needed by ResultFilterBar's op dropdown)
  http.get('/api/v1/workflows/:workflowId/ops', ({ params }) => {
    return HttpResponse.json({
      ops: STORY_RESULTS.map((r) => ({
        op: makeWorkflowOp({ ID: r.opID, Kind: r.kind }),
        status: r.status,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      })),
    });
  }),

  // Full op result body for ResultPreviewPanel (extract op — JSON records)
  http.get(`/api/v1/workflows/${STORY_WORKFLOW_ID}/ops/${STORY_WORKFLOW_ID}:extract/result`, () => {
    return HttpResponse.json({
      result: {
        OpID: `${STORY_WORKFLOW_ID}:extract`,
        Data: {
          stories: [
            { id: 12345, url: 'https://example.com/1', title: 'Show HN: A cool project' },
            { id: 12346, url: 'https://example.com/2', title: 'Another story' },
            { id: 12347, url: 'https://example.com/3', title: 'Yet another story' },
          ],
          nextPage: '/news?p=2',
          scrapedAt: new Date().toISOString(),
        },
        Records: [
          { Collection: 'items', Key: '12345', Data: { id: 12345, title: 'A cool project' } },
          { Collection: 'items', Key: '12346', Data: { id: 12346, title: 'Another story' } },
        ],
        Artifacts: [],
        Emitted: [],
        EmittedIDs: [],
        Error: undefined,
        CompletedAt: new Date().toISOString(),
      },
    });
  }),

  // Full op result body for failed op
  http.get(`/api/v1/workflows/${STORY_WORKFLOW_ID}/ops/${STORY_WORKFLOW_ID}:page-2-extract/result`, () => {
    return HttpResponse.json({
      result: {
        OpID: `${STORY_WORKFLOW_ID}:page-2-extract`,
        Data: null,
        Records: [],
        Artifacts: [],
        Emitted: [],
        EmittedIDs: [],
        Error: {
          Code: 'JSError',
          Message: 'SyntaxError: Unexpected token at line 12',
          Retryable: false,
          Details: null,
          OccurredAt: new Date().toISOString(),
        },
        CompletedAt: new Date().toISOString(),
      },
    });
  }),
];

// ─── Empty results list handler ──────────────────────────────────────────────

export const emptyResultsHandlers = [
  http.get('/api/v1/workflows/:workflowId/results', ({ params }) => {
    return HttpResponse.json({
      workflowID: String(params.workflowId),
      total: 0,
      results: [],
    });
  }),
  http.get('/api/v1/workflows/:workflowId/ops', ({ params }) => {
    return HttpResponse.json({ ops: [] });
  }),
];
