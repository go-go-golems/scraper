import type { Meta, StoryObj } from '@storybook/react';
import { OpDepsTab } from './OpDepsTab';

const meta: Meta<typeof OpDepsTab> = {
  title: 'Workflows/OpDetail/OpDepsTab',
  component: OpDepsTab,
};

export default meta;
type Story = StoryObj<typeof OpDepsTab>;

export const NoDeps: Story = {
  args: {
    dependsOn: [],
  },
};

export const SingleRequiredDep: Story = {
  args: {
    dependsOn: [{ OpID: 'wf-001:fetch-1', Required: true }],
  },
};

export const MultipleDeps: Story = {
  args: {
    dependsOn: [
      { OpID: 'wf-001:fetch-1', Required: true },
      { OpID: 'wf-001:fetch-2', Required: true },
      { OpID: 'wf-001:fetch-3', Required: false },
    ],
  },
};

export const ManyDeps: Story = {
  args: {
    dependsOn: Array.from({ length: 12 }, (_, i) => ({
      OpID: `wf-001:fetch-${i + 1}`,
      Required: i < 3,
    })),
  },
};
