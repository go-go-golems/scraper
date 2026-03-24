import type { Meta, StoryObj } from '@storybook/react';
import { OpExecutionLog } from './OpExecutionLog';
import { createLogEntries } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof OpExecutionLog> = {
  title: 'Logs/OpExecutionLog',
  component: OpExecutionLog,
  args: {
    loading: false,
  },
};

export default meta;
type Story = StoryObj<typeof OpExecutionLog>;

export const Default: Story = {
  args: {
    entries: createLogEntries(15),
  },
};

export const Empty: Story = {
  args: {
    entries: [],
  },
};

export const Long: Story = {
  args: {
    entries: createLogEntries(200),
  },
};

export const Loading: Story = {
  args: {
    entries: [],
    loading: true,
  },
};
