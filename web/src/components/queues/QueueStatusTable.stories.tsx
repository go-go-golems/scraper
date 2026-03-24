import type { Meta, StoryObj } from '@storybook/react-vite';
import { QueueStatusTable } from './QueueStatusTable';
import { createQueueStatus } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof QueueStatusTable> = {
  title: 'Queues/QueueStatusTable',
  component: QueueStatusTable,
};

export default meta;
type Story = StoryObj<typeof QueueStatusTable>;

export const Default: Story = {
  args: {
    loading: false,
    queues: [
      createQueueStatus({
        site: 'hackernews',
        queue: 'site:hn:http',
        inFlight: 2,
        maxInFlight: 4,
        tokens: 10,
        ratePerSecond: 2,
        burst: 5,
      }),
      createQueueStatus({
        site: 'hackernews',
        queue: 'site:hn:js',
        inFlight: 1,
        maxInFlight: 4,
      }),
      createQueueStatus({
        site: 'slashdot',
        queue: 'site:sd:http',
        inFlight: 3,
        maxInFlight: 4,
        tokens: 8,
        ratePerSecond: 1,
        burst: 3,
      }),
      createQueueStatus({
        site: 'slashdot',
        queue: 'site:sd:js',
        inFlight: 0,
        maxInFlight: 2,
      }),
      createQueueStatus({
        site: 'nereval',
        queue: 'site:nv:http',
        inFlight: 4,
        maxInFlight: 8,
        tokens: 20,
        ratePerSecond: 5,
        burst: 10,
      }),
      createQueueStatus({
        site: 'nereval',
        queue: 'site:nv:js',
        inFlight: 0,
        maxInFlight: 1,
      }),
    ],
  },
};

export const AllIdle: Story = {
  args: {
    loading: false,
    queues: [
      createQueueStatus({ queue: 'site:hn:http', inFlight: 0, maxInFlight: 4 }),
      createQueueStatus({ queue: 'site:hn:js', inFlight: 0, maxInFlight: 4 }),
      createQueueStatus({ site: 'slashdot', queue: 'site:sd:http', inFlight: 0, maxInFlight: 4 }),
      createQueueStatus({ site: 'slashdot', queue: 'site:sd:js', inFlight: 0, maxInFlight: 2 }),
    ],
  },
};

export const Saturated: Story = {
  args: {
    loading: false,
    queues: [
      createQueueStatus({
        queue: 'site:hn:http',
        inFlight: 4,
        maxInFlight: 4,
        tokens: 0,
        ratePerSecond: 2,
        burst: 5,
      }),
      createQueueStatus({
        queue: 'site:hn:js',
        inFlight: 4,
        maxInFlight: 4,
      }),
      createQueueStatus({
        site: 'slashdot',
        queue: 'site:sd:http',
        inFlight: 8,
        maxInFlight: 8,
        tokens: 0,
        ratePerSecond: 1,
        burst: 3,
      }),
      createQueueStatus({
        site: 'nereval',
        queue: 'site:nv:http',
        inFlight: 2,
        maxInFlight: 2,
        tokens: 0,
        ratePerSecond: 5,
        burst: 10,
      }),
    ],
  },
};

export const NoRateLimit: Story = {
  args: {
    loading: false,
    queues: [
      createQueueStatus({ queue: 'site:hn:http', inFlight: 2, maxInFlight: 4 }),
      createQueueStatus({ queue: 'site:hn:js', inFlight: 1, maxInFlight: 4 }),
      createQueueStatus({ site: 'slashdot', queue: 'site:sd:http', inFlight: 0, maxInFlight: 4 }),
    ],
  },
};

export const Loading: Story = {
  args: {
    loading: true,
    queues: [],
  },
};

export const Empty: Story = {
  args: {
    loading: false,
    queues: [],
  },
};
