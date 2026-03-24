import type { Meta, StoryObj } from '@storybook/react';
import { ConfirmDialog } from './ConfirmDialog';

const meta: Meta<typeof ConfirmDialog> = {
  title: 'Common/ConfirmDialog',
  component: ConfirmDialog,
};

export default meta;
type Story = StoryObj<typeof ConfirmDialog>;

export const CancelWorkflow: Story = {
  args: {
    open: true,
    title: 'Cancel Workflow',
    message: 'Are you sure you want to cancel this workflow? This action cannot be undone.',
    confirmLabel: 'Cancel Workflow',
    confirmColor: 'error',
    loading: false,
    onConfirm: () => {},
    onCancel: () => {},
  },
};

export const RetryOp: Story = {
  args: {
    open: true,
    title: 'Retry Operation',
    message: 'This will retry the failed operation from the beginning.',
    confirmLabel: 'Retry',
    confirmColor: 'primary',
    loading: false,
    onConfirm: () => {},
    onCancel: () => {},
  },
};

export const Loading: Story = {
  args: {
    open: true,
    title: 'Cancel Workflow',
    message: 'Are you sure you want to cancel this workflow? This action cannot be undone.',
    confirmLabel: 'Cancel Workflow',
    confirmColor: 'error',
    loading: true,
    onConfirm: () => {},
    onCancel: () => {},
  },
};
