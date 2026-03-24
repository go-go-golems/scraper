import type { Meta, StoryObj } from '@storybook/react';
import { StatusChip } from './StatusChip';
import { Box } from '@mui/material';

const meta: Meta<typeof StatusChip> = {
  title: 'Common/StatusChip',
  component: StatusChip,
};

export default meta;
type Story = StoryObj<typeof StatusChip>;

export const Pending: Story = { args: { status: 'pending' } };
export const Ready: Story = { args: { status: 'ready' } };
export const Running: Story = { args: { status: 'running' } };
export const Succeeded: Story = { args: { status: 'succeeded' } };
export const Failed: Story = { args: { status: 'failed' } };
export const Canceled: Story = { args: { status: 'canceled' } };

export const AllStatuses: Story = {
  render: () => (
    <Box sx={{ display: 'flex', gap: 1 }}>
      <StatusChip status="pending" />
      <StatusChip status="ready" />
      <StatusChip status="running" />
      <StatusChip status="succeeded" />
      <StatusChip status="failed" />
      <StatusChip status="canceled" />
    </Box>
  ),
};
