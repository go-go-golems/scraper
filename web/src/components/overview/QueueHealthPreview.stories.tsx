import type { Meta, StoryObj } from '@storybook/react';
import { QueueHealthPreview } from './QueueHealthPreview';
import { createQueueStatus } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof QueueHealthPreview> = {
  title: 'Overview/QueueHealthPreview',
  component: QueueHealthPreview,
};

export default meta;
type Story = StoryObj<typeof QueueHealthPreview>;

export const Default: Story = {
  args: {
    queues: [
      createQueueStatus({ queue: 'site:hn:http', inFlight: 2, maxInFlight: 4 }),
      createQueueStatus({ queue: 'site:hn:js', inFlight: 1, maxInFlight: 4 }),
      createQueueStatus({ site: 'slashdot', queue: 'site:sd:http', inFlight: 0, maxInFlight: 4 }),
      createQueueStatus({ site: 'nereval', queue: 'site:nv:js', inFlight: 0, maxInFlight: 1 }),
    ],
  },
};

export const Saturated: Story = {
  args: {
    queues: [
      createQueueStatus({ queue: 'site:hn:http', inFlight: 4, maxInFlight: 4 }),
      createQueueStatus({ queue: 'site:hn:js', inFlight: 4, maxInFlight: 4 }),
      createQueueStatus({ site: 'nereval', queue: 'site:nv:http', inFlight: 4, maxInFlight: 4 }),
    ],
  },
};

export const AllIdle: Story = {
  args: {
    queues: [
      createQueueStatus({ queue: 'site:hn:http', inFlight: 0, maxInFlight: 4 }),
      createQueueStatus({ queue: 'site:hn:js', inFlight: 0, maxInFlight: 4 }),
    ],
  },
};

export const Empty: Story = {
  args: { queues: [] },
};
