import type { Meta, StoryObj } from '@storybook/react';
import { WorkflowTable } from './WorkflowTable';
import { createWorkflowListItem } from '../../stories/__fixtures__/factories';
import type { WorkflowStatus } from '../../api/types';

const meta: Meta<typeof WorkflowTable> = {
  title: 'Workflows/WorkflowTable',
  component: WorkflowTable,
};

export default meta;
type Story = StoryObj<typeof WorkflowTable>;

const statuses: WorkflowStatus[] = ['pending', 'running', 'succeeded', 'failed', 'canceled'];
const sites = ['hackernews', 'slashdot', 'js-demo', 'nereval'];

const mixedWorkflows = Array.from({ length: 10 }, (_, i) =>
  createWorkflowListItem({
    id: `wf-${String(i + 1).padStart(3, '0')}`,
    site: sites[i % sites.length],
    status: statuses[i % statuses.length],
    opTotal: 20 + i * 5,
    opDone: Math.min(10 + i * 3, 20 + i * 5),
  }),
);

export const Default: Story = {
  args: {
    workflows: mixedWorkflows,
    loading: false,
    onWorkflowClick: () => {},
  },
};

export const Empty: Story = {
  args: {
    workflows: [],
    loading: false,
    onWorkflowClick: () => {},
  },
};

export const Loading: Story = {
  args: {
    workflows: [],
    loading: true,
    onWorkflowClick: () => {},
  },
};

export const SingleSite: Story = {
  args: {
    workflows: Array.from({ length: 5 }, (_, i) =>
      createWorkflowListItem({
        id: `wf-hn-${String(i + 1).padStart(3, '0')}`,
        site: 'hackernews',
        status: statuses[i % statuses.length],
        opTotal: 47,
        opDone: i * 10,
      }),
    ),
    loading: false,
    onWorkflowClick: () => {},
  },
};
