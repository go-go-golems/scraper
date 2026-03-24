import type { Meta, StoryObj } from '@storybook/react';
import { RetryOpButton } from './RetryOpButton';

const meta: Meta<typeof RetryOpButton> = {
  title: 'Workflows/RetryOpButton',
  component: RetryOpButton,
};

export default meta;
type Story = StoryObj<typeof RetryOpButton>;

export const Default: Story = {
  args: {
    workflowId: 'wf-001',
    opId: 'wf-001:seed',
    disabled: false,
    loading: false,
    onRetry: () => {},
  },
};

export const Disabled: Story = {
  args: {
    workflowId: 'wf-001',
    opId: 'wf-001:seed',
    disabled: true,
    loading: false,
    onRetry: () => {},
  },
};

export const Loading: Story = {
  args: {
    workflowId: 'wf-001',
    opId: 'wf-001:seed',
    disabled: false,
    loading: true,
    onRetry: () => {},
  },
};
