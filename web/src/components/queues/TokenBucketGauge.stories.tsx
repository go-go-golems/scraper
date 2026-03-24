import type { Meta, StoryObj } from '@storybook/react-vite';
import { TokenBucketGauge } from './TokenBucketGauge';

const meta: Meta<typeof TokenBucketGauge> = {
  title: 'Queues/TokenBucketGauge',
  component: TokenBucketGauge,
};

export default meta;
type Story = StoryObj<typeof TokenBucketGauge>;

export const Full: Story = {
  args: {
    tokens: 4.0,
    burst: 4.0,
    ratePerSecond: 2.0,
  },
};

export const Depleted: Story = {
  args: {
    tokens: 0.2,
    burst: 4.0,
    ratePerSecond: 2.0,
  },
};

export const Half: Story = {
  args: {
    tokens: 1.8,
    burst: 4.0,
    ratePerSecond: 2.0,
  },
};

export const NoRateLimit: Story = {
  args: {
    tokens: 0,
    burst: 0,
    ratePerSecond: 0,
  },
};
