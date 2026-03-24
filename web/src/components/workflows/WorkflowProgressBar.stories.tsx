import type { Meta, StoryObj } from '@storybook/react';
import { WorkflowProgressBar } from './WorkflowProgressBar';

const meta: Meta<typeof WorkflowProgressBar> = {
  title: 'Workflows/WorkflowProgressBar',
  component: WorkflowProgressBar,
};

export default meta;
type Story = StoryObj<typeof WorkflowProgressBar>;

export const InProgress: Story = {
  args: {
    stats: {
      WorkflowID: 'wf-001',
      Total: 47,
      Pending: 5,
      Ready: 8,
      Running: 3,
      Succeeded: 28,
      Failed: 2,
      Canceled: 1,
    },
  },
};

export const Complete: Story = {
  args: {
    stats: {
      WorkflowID: 'wf-002',
      Total: 30,
      Pending: 0,
      Ready: 0,
      Running: 0,
      Succeeded: 30,
      Failed: 0,
      Canceled: 0,
    },
  },
};

export const HasFailures: Story = {
  args: {
    stats: {
      WorkflowID: 'wf-003',
      Total: 20,
      Pending: 0,
      Ready: 0,
      Running: 0,
      Succeeded: 14,
      Failed: 5,
      Canceled: 1,
    },
  },
};
