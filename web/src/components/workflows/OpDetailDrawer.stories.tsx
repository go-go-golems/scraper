import type { Meta, StoryObj } from '@storybook/react';
import { OpDetailDrawer } from './OpDetailDrawer';
import { createWorkflowOp, createOpResult } from '../../stories/__fixtures__/factories';

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
    open: true,
    onClose: () => {},
  },
};

export const HttpOpSucceeded: Story = {
  args: {
    op: createWorkflowOp({ id: 'wf-001:fetch-1', kind: 'http', status: 'succeeded', queue: 'site:hackernews:http' }),
    result: createOpResult({
      opId: 'wf-001:fetch-1',
      data: { statusCode: 200, bodyLength: 12345 },
      artifacts: [{ id: 'art-001', name: 'page.html', kind: 'html', contentType: 'text/html' }],
    }),
    open: true,
    onClose: () => {},
  },
};

export const OpFailedRetryable: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:fetch-err', kind: 'http', status: 'failed' }),
      op: {
        ...createWorkflowOp({ id: 'wf-001:fetch-err', kind: 'http', status: 'failed' }).op,
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
  },
};

export const OpRunning: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:fetch-active', kind: 'http', status: 'running' }),
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
