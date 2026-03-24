import type { Meta, StoryObj } from '@storybook/react';
import { OpDetailDrawer } from './OpDetailDrawer';
import { createWorkflowOp, createOpResult, createArtifactSummary } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof OpDetailDrawer> = {
  title: 'Workflows/OpDetailDrawer',
  component: OpDetailDrawer,
  parameters: {
    layout: 'fullscreen',
  },
};

export default meta;
type Story = StoryObj<typeof OpDetailDrawer>;

export const JsOpSucceeded: Story = {
  args: {
    op: createWorkflowOp({ id: 'wf-001:seed', kind: 'js', status: 'succeeded' }),
    result: createOpResult({
      opId: 'wf-001:seed',
      data: { urls: ['https://news.ycombinator.com/item?id=1', 'https://news.ycombinator.com/item?id=2'] },
      emittedIds: ['wf-001:fetch-1', 'wf-001:fetch-2', 'wf-001:fetch-3'],
    }),
    artifacts: [
      createArtifactSummary({ id: 'log-1', name: 'execution-log', kind: 'execution-log', contentType: 'application/json', size: 512 }),
    ],
    artifactBodies: {
      'log-1': JSON.stringify([
        { timestamp: '2026-03-23T14:32:01.234Z', message: 'Starting seed workflow' },
        { timestamp: '2026-03-23T14:32:01.236Z', message: 'Emitting 3 fetch ops' },
        { timestamp: '2026-03-23T14:32:01.237Z', message: 'Done' },
      ]),
    },
    scriptSource: 'const helpers = require("./lib/frontpage");\n\nmodule.exports = function(ctx) {\n  const input = ctx.input;\n  ctx.log("Starting seed workflow");\n  // emit fetch ops\n  for (let i = 0; i < input.maxPages; i++) {\n    ctx.emit({\n      kind: "http/fetch",\n      queue: "site:hackernews:http",\n      input: { request: { url: input.baseURL } }\n    });\n  }\n  ctx.log("Emitting " + input.maxPages + " fetch ops");\n  ctx.log("Done");\n};',
    open: true,
    onClose: () => {},
  },
};

export const WithArtifacts: Story = {
  args: {
    op: createWorkflowOp({ id: 'wf-001:fetch-1', kind: 'http/fetch', status: 'succeeded', queue: 'site:hackernews:http' }),
    result: createOpResult({
      opId: 'wf-001:fetch-1',
      data: { statusCode: 200, bodyLength: 12345 },
      artifacts: [{ id: 'art-001', name: 'frontpage.html', kind: 'html', contentType: 'text/html' }],
    }),
    artifacts: [
      createArtifactSummary({ id: 'art-001', name: 'frontpage.html', kind: 'html', contentType: 'text/html', size: 12345 }),
      createArtifactSummary({ id: 'art-002', name: 'headers.json', kind: 'json', contentType: 'application/json', size: 842 }),
    ],
    artifactBodies: {
      'art-001': '<!DOCTYPE html>\n<html>\n<head><title>Hacker News</title></head>\n<body>\n<table>\n  <tr><td>1.</td><td><a href="...">Show HN: Something cool</a></td></tr>\n</table>\n</body>\n</html>',
      'art-002': '{\n  "Content-Type": "text/html; charset=utf-8",\n  "Server": "nginx"\n}',
    },
    open: true,
    onClose: () => {},
  },
};

export const OpFailedRetryable: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:fetch-err', kind: 'http/fetch', status: 'failed' }),
      op: {
        ...createWorkflowOp({ id: 'wf-001:fetch-err', kind: 'http/fetch', status: 'failed' }).op,
        RetryState: { Attempt: 3, LastError: 'rate limited' },
      },
    },
    result: createOpResult({
      opId: 'wf-001:fetch-err',
      error: {
        code: 'HTTP_429',
        message: 'Rate limited by upstream server. Retry-After: 60s',
        retryable: true,
      },
    }),
    open: true,
    onClose: () => {},
    onRetry: () => {},
  },
};

export const OpRunning: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:fetch-active', kind: 'http/fetch', status: 'running' }),
      lease: {
        WorkerID: 'worker-alpha-01',
        Token: 'lease-tok-abc123',
        AcquiredAt: '2026-03-23T14:32:00Z',
        ExpiresAt: '2026-03-23T14:33:00Z',
      },
    },
    result: null,
    open: true,
    onClose: () => {},
  },
};

export const OpPending: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:parse-1', kind: 'js', status: 'pending' }),
      op: {
        ...createWorkflowOp({ id: 'wf-001:parse-1', kind: 'js', status: 'pending' }).op,
        DependsOn: [
          { OpID: 'wf-001:fetch-1', Required: true },
          { OpID: 'wf-001:fetch-2', Required: false },
        ],
      },
    },
    result: null,
    open: true,
    onClose: () => {},
  },
};

export const WithLogs: Story = {
  args: {
    op: createWorkflowOp({ id: 'wf-001:extract', kind: 'js', status: 'succeeded' }),
    result: createOpResult({ opId: 'wf-001:extract', data: { stories: 30 } }),
    artifacts: [
      createArtifactSummary({ id: 'log-1', name: 'execution-log', kind: 'execution-log', contentType: 'application/json', size: 1024 }),
    ],
    artifactBodies: {
      'log-1': JSON.stringify(
        Array.from({ length: 15 }, (_, i) => ({
          timestamp: `2026-03-23T14:32:0${i}.${String(i * 37).padStart(3, '0')}Z`,
          message: [
            'Parsing frontpage HTML',
            'Found 30 stories',
            'Extracting story: Show HN: New framework',
            'Extracting story: Ask HN: Best tools',
            'Extracting story: Launch HN: Our startup',
            'Writing 30 rows to site DB',
            'Checking for next page link',
            'Found next page URL',
            'Emitting page-2 fetch op',
            'Emitting page-2 extract op',
            'Setting up dependency chain',
            'Writing summary artifact',
            'All stories extracted',
            'Cleaning up temporary data',
            'Done. 30 stories, 2 child ops',
          ][i],
        })),
      ),
    },
    open: true,
    onClose: () => {},
  },
};
