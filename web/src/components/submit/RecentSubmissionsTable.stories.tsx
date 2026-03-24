import type { Meta, StoryObj } from '@storybook/react';
import { RecentSubmissionsTable } from './RecentSubmissionsTable';

const meta: Meta<typeof RecentSubmissionsTable> = {
  title: 'Submit/RecentSubmissionsTable',
  component: RecentSubmissionsTable,
};

export default meta;
type Story = StoryObj<typeof RecentSubmissionsTable>;

const now = new Date();

export const Default: Story = {
  args: {
    submissions: [
      {
        timestamp: new Date(now.getTime() - 30_000).toISOString(),
        site: 'hackernews',
        verb: 'seed',
        workflowId: 'wf-abc-001',
      },
      {
        timestamp: new Date(now.getTime() - 120_000).toISOString(),
        site: 'js-demo',
        verb: 'seed',
        workflowId: 'wf-def-002',
      },
      {
        timestamp: new Date(now.getTime() - 3_600_000).toISOString(),
        site: 'nereval',
        verb: 'seed',
        workflowId: 'wf-ghi-003',
      },
    ],
  },
};

export const Empty: Story = {
  args: {
    submissions: [],
  },
};
