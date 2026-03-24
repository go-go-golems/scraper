import type { Meta, StoryObj } from '@storybook/react-vite';
import { QueueDetailPanel } from './QueueDetailPanel';
import { createQueueStatus } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof QueueDetailPanel> = {
  title: 'Queues/QueueDetailPanel',
  component: QueueDetailPanel,
};

export default meta;
type Story = StoryObj<typeof QueueDetailPanel>;

export const WithRateLimit: Story = {
  args: {
    queue: createQueueStatus({
      queue: 'site:hn:http',
      inFlight: 2,
      maxInFlight: 4,
      tokens: 1.8,
      burst: 4,
      ratePerSecond: 2,
      pending: 3,
      ready: 5,
      running: 2,
      succeeded: 120,
      failed: 1,
    }),
  },
};

export const WithoutRateLimit: Story = {
  args: {
    queue: createQueueStatus({
      queue: 'site:hn:js',
      inFlight: 1,
      maxInFlight: 4,
      pending: 0,
      ready: 2,
      running: 1,
      succeeded: 80,
      failed: 0,
    }),
  },
};

export const Saturated: Story = {
  args: {
    queue: createQueueStatus({
      queue: 'site:sd:http',
      inFlight: 8,
      maxInFlight: 8,
      tokens: 0,
      burst: 3,
      ratePerSecond: 1,
      pending: 12,
      ready: 0,
      running: 8,
      succeeded: 200,
      failed: 5,
    }),
  },
};

export const Idle: Story = {
  args: {
    queue: createQueueStatus({
      queue: 'site:nv:http',
      inFlight: 0,
      maxInFlight: 4,
      tokens: 10,
      burst: 10,
      ratePerSecond: 5,
      pending: 0,
      ready: 0,
      running: 0,
      succeeded: 50,
      failed: 0,
    }),
  },
};
