import type { Meta, StoryObj } from '@storybook/react';
import { StatCard } from './StatCard';

const meta: Meta<typeof StatCard> = {
  title: 'Overview/StatCard',
  component: StatCard,
};

export default meta;
type Story = StoryObj<typeof StatCard>;

export const Workflows: Story = {
  args: {
    title: 'Workflows',
    value: 12,
    breakdown: [
      { label: 'running', value: 3, color: 'info' },
      { label: 'succeeded', value: 8, color: 'success' },
      { label: 'failed', value: 1, color: 'error' },
    ],
  },
};

export const Operations: Story = {
  args: {
    title: 'Operations',
    value: 847,
    breakdown: [
      { label: 'ready', value: 23, color: 'info' },
      { label: 'running', value: 4, color: 'primary' },
      { label: 'failed', value: 12, color: 'error' },
    ],
  },
};

export const SimpleCount: Story = {
  args: {
    title: 'Artifacts',
    value: 1234,
  },
};

export const ZeroState: Story = {
  args: {
    title: 'Workflows',
    value: 0,
    breakdown: [],
  },
};

export const HighCounts: Story = {
  args: {
    title: 'Operations',
    value: 125847,
    breakdown: [
      { label: 'ready', value: 2300, color: 'info' },
      { label: 'running', value: 48, color: 'primary' },
      { label: 'failed', value: 127, color: 'error' },
    ],
  },
};

export const Loading: Story = {
  args: {
    title: '',
    value: 0,
    loading: true,
  },
};
