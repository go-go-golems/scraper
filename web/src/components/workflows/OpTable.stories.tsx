import type { Meta, StoryObj } from '@storybook/react';
import { OpTable } from './OpTable';
import { createWorkflowOp } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof OpTable> = {
  title: 'Workflows/OpTable',
  component: OpTable,
};

export default meta;
type Story = StoryObj<typeof OpTable>;

export const Default: Story = {
  args: {
    ops: [
      createWorkflowOp({ id: 'wf-001:seed', kind: 'js', status: 'succeeded' }),
      createWorkflowOp({ id: 'wf-001:fetch-1', kind: 'http', status: 'running', queue: 'site:hackernews:http' }),
      createWorkflowOp({ id: 'wf-001:fetch-2', kind: 'http', status: 'ready', queue: 'site:hackernews:http' }),
      createWorkflowOp({ id: 'wf-001:parse-1', kind: 'js', status: 'pending' }),
      createWorkflowOp({ id: 'wf-001:fetch-3', kind: 'http', status: 'failed', queue: 'site:hackernews:http' }),
    ],
    selectedOpId: null,
    onSelectOp: () => {},
  },
};

export const WithRetries: Story = {
  args: {
    ops: [
      {
        ...createWorkflowOp({ id: 'wf-001:flaky-1', kind: 'http', status: 'running' }),
        op: {
          ...createWorkflowOp({ id: 'wf-001:flaky-1', kind: 'http', status: 'running' }).op,
          RetryState: { Attempt: 2, LastError: 'connection timeout' },
        },
      },
      {
        ...createWorkflowOp({ id: 'wf-001:flaky-2', kind: 'http', status: 'failed' }),
        op: {
          ...createWorkflowOp({ id: 'wf-001:flaky-2', kind: 'http', status: 'failed' }).op,
          RetryState: { Attempt: 3, LastError: 'rate limited' },
        },
      },
      createWorkflowOp({ id: 'wf-001:ok-1', kind: 'js', status: 'succeeded' }),
    ],
    selectedOpId: null,
    onSelectOp: () => {},
  },
};

export const AllSucceeded: Story = {
  args: {
    ops: Array.from({ length: 6 }, (_, i) =>
      createWorkflowOp({
        id: `wf-001:op-${i + 1}`,
        kind: i % 2 === 0 ? 'js' : 'http',
        status: 'succeeded',
      }),
    ),
    selectedOpId: null,
    onSelectOp: () => {},
  },
};
