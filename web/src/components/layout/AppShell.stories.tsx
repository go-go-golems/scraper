import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { AppShell } from './AppShell';
import { Typography } from '@mui/material';

const meta: Meta<typeof AppShell> = {
  title: 'Layout/AppShell',
  component: AppShell,
  args: {
    children: <Typography sx={{ p: 4, color: 'text.secondary' }}>Page content goes here</Typography>,
  },
};

export default meta;
type Story = StoryObj<typeof AppShell>;

export const OverviewTab: Story = {
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={['/']}>
        <Story />
      </MemoryRouter>
    ),
  ],
};

export const WorkflowsTab: Story = {
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={['/workflows']}>
        <Story />
      </MemoryRouter>
    ),
  ],
};

export const QueuesTab: Story = {
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={['/queues']}>
        <Story />
      </MemoryRouter>
    ),
  ],
};

export const SubmitTab: Story = {
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={['/submit']}>
        <Story />
      </MemoryRouter>
    ),
  ],
};
