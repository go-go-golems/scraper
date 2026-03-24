import type { Meta, StoryObj } from '@storybook/react';
import { CancelWorkflowButton } from './CancelWorkflowButton';

const meta: Meta<typeof CancelWorkflowButton> = {
  title: 'Workflows/CancelWorkflowButton',
  component: CancelWorkflowButton,
};

export default meta;
type Story = StoryObj<typeof CancelWorkflowButton>;

export const Running: Story = {
  args: {
    workflowId: 'wf-001',
    status: 'running',
    loading: false,
    onCancel: () => {},
  },
};

export const Succeeded: Story = {
  args: {
    workflowId: 'wf-001',
    status: 'succeeded',
    loading: false,
    onCancel: () => {},
  },
};

export const Loading: Story = {
  args: {
    workflowId: 'wf-001',
    status: 'running',
    loading: true,
    onCancel: () => {},
  },
};
