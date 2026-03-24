import type { Meta, StoryObj } from '@storybook/react';
import { WorkflowHeader } from './WorkflowHeader';
import { createWorkflowSummary } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof WorkflowHeader> = {
  title: 'Workflows/WorkflowHeader',
  component: WorkflowHeader,
};

export default meta;
type Story = StoryObj<typeof WorkflowHeader>;

export const Running: Story = {
  args: {
    workflow: createWorkflowSummary({ status: 'running' }),
  },
};

export const Succeeded: Story = {
  args: {
    workflow: createWorkflowSummary({ status: 'succeeded' }),
  },
};

export const Failed: Story = {
  args: {
    workflow: createWorkflowSummary({ status: 'failed' }),
  },
};
