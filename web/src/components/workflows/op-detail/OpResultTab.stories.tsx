import type { Meta, StoryObj } from '@storybook/react';
import { OpResultTab } from './OpResultTab';
import { createWorkflowOp, createOpResult } from '../../../stories/__fixtures__/factories';

const meta: Meta<typeof OpResultTab> = {
  title: 'Workflows/OpDetail/OpResultTab',
  component: OpResultTab,
};

export default meta;
type Story = StoryObj<typeof OpResultTab>;

export const SucceededWithData: Story = {
  args: {
    op: createWorkflowOp({ id: 'wf-001:seed', kind: 'js', status: 'succeeded' }),
    result: createOpResult({
      opId: 'wf-001:seed',
      data: { urls: ['https://news.ycombinator.com/item?id=1', 'https://news.ycombinator.com/item?id=2'] },
      emittedIds: ['wf-001:fetch-1', 'wf-001:fetch-2', 'wf-001:fetch-3'],
    }),
  },
};

export const NoResultYet: Story = {
  args: {
    op: createWorkflowOp({ id: 'wf-001:pending-op', kind: 'js', status: 'pending' }),
    result: null,
  },
};

export const FailedWithRetryableError: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:fetch-err', kind: 'http/fetch', status: 'failed' }),
      op: {
        ...createWorkflowOp({ id: 'wf-001:fetch-err', kind: 'http/fetch', status: 'failed' }).op,
        RetryState: { Attempt: 3, LastError: 'rate limited by upstream' },
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
  },
};

export const FailedNonRetryable: Story = {
  args: {
    op: {
      ...createWorkflowOp({ id: 'wf-001:parse-err', kind: 'js', status: 'failed' }),
      op: {
        ...createWorkflowOp({ id: 'wf-001:parse-err', kind: 'js', status: 'failed' }).op,
        RetryState: { Attempt: 3, LastError: 'syntax error in script' },
      },
    },
    result: createOpResult({
      opId: 'wf-001:parse-err',
      data: { partialResult: 'some data' },
      error: {
        code: 'SCRIPT_ERROR',
        message: 'TypeError: Cannot read property "title" of undefined at line 13',
        retryable: false,
      },
    }),
  },
};

export const RunningWithLease: Story = {
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
  },
};
