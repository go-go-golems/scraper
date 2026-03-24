import type { Meta, StoryObj } from '@storybook/react';
import { OpStatusBreakdown } from './OpStatusBreakdown';

const meta: Meta<typeof OpStatusBreakdown> = {
  title: 'Overview/OpStatusBreakdown',
  component: OpStatusBreakdown,
};

export default meta;
type Story = StoryObj<typeof OpStatusBreakdown>;

export const Default: Story = {
  args: {
    counts: { pending: 3, ready: 23, running: 4, succeeded: 808, failed: 12, canceled: 0 },
  },
};

export const AllSucceeded: Story = {
  args: {
    counts: { pending: 0, ready: 0, running: 0, succeeded: 500, failed: 0, canceled: 0 },
  },
};

export const MostlyPending: Story = {
  args: {
    counts: { pending: 200, ready: 15, running: 3, succeeded: 12, failed: 0, canceled: 0 },
  },
};

export const HasFailures: Story = {
  args: {
    counts: { pending: 0, ready: 2, running: 1, succeeded: 40, failed: 15, canceled: 0 },
  },
};

export const SmallWorkflow: Story = {
  args: {
    counts: { pending: 0, ready: 0, running: 1, succeeded: 3, failed: 0, canceled: 0 },
  },
};
