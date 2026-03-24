import type { Meta, StoryObj } from '@storybook/react';
import { StatCardRow } from './StatCardRow';
import { createEngineStatus, createEmptyEngineStatus } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof StatCardRow> = {
  title: 'Overview/StatCardRow',
  component: StatCardRow,
};

export default meta;
type Story = StoryObj<typeof StatCardRow>;

export const Default: Story = {
  args: { status: createEngineStatus() },
};

export const Empty: Story = {
  args: { status: createEmptyEngineStatus() },
};

export const Loading: Story = {
  args: { loading: true },
};

export const HighActivity: Story = {
  args: {
    status: createEngineStatus({
      WorkflowCount: 87,
      OpCounts: { pending: 120, ready: 340, running: 24, succeeded: 12400, failed: 89, canceled: 3 },
      ActiveLeases: 24,
      ExpiredLeases: 2,
      ArtifactCount: 8900,
    }),
  },
};
