import type { Meta, StoryObj } from '@storybook/react';
import { AppShell } from './AppShell';
import { Typography } from '@mui/material';

const meta: Meta<typeof AppShell> = {
  title: 'Layout/AppShell',
  component: AppShell,
  args: {
    currentTab: 'overview',
    onTabChange: () => {},
    children: <Typography sx={{ p: 4, color: 'text.secondary' }}>Page content goes here</Typography>,
  },
};

export default meta;
type Story = StoryObj<typeof AppShell>;

export const Default: Story = {};

export const WorkflowsTab: Story = {
  args: { currentTab: 'workflows' },
};

export const QueuesTab: Story = {
  args: { currentTab: 'queues' },
};

export const SubmitTab: Story = {
  args: { currentTab: 'submit' },
};
