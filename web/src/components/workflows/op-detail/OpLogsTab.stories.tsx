import type { Meta, StoryObj } from '@storybook/react';
import { OpLogsTab } from './OpLogsTab';
import { createLogEntries } from '../../../stories/__fixtures__/factories';

const meta: Meta<typeof OpLogsTab> = {
  title: 'Workflows/OpDetail/OpLogsTab',
  component: OpLogsTab,
};

export default meta;
type Story = StoryObj<typeof OpLogsTab>;

export const NoLogs: Story = {
  args: {
    entries: [],
  },
};

export const FewEntries: Story = {
  args: {
    entries: [
      { timestamp: '2026-03-23T14:32:01.234Z', message: 'Starting seed workflow' },
      { timestamp: '2026-03-23T14:32:01.236Z', message: 'Emitting 3 fetch ops' },
      { timestamp: '2026-03-23T14:32:01.237Z', message: 'Done' },
    ],
  },
};

export const ManyEntries: Story = {
  args: {
    entries: createLogEntries(20),
  },
};
