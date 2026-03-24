import type { Meta, StoryObj } from '@storybook/react';
import { SiteCard } from './SiteCard';
import { createSiteDetail } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof SiteCard> = {
  title: 'Sites/SiteCard',
  component: SiteCard,
};

export default meta;
type Story = StoryObj<typeof SiteCard>;

export const Default: Story = {
  args: {
    site: createSiteDetail('hackernews'),
    onClick: () => {},
  },
};

export const WithQueuePolicies: Story = {
  args: {
    site: createSiteDetail('nereval', {
      verbCount: 3,
      scriptCount: 5,
      scripts: ['seed.js', 'detail.js', 'lib/utils.js', 'lib/parse.js', 'export.js'],
      queuePolicies: [
        {
          queue: 'site:nereval:http',
          maxInFlight: 8,
          rateLimit: { kind: 'token-bucket', ratePerSecond: 5, burst: 10 },
        },
        {
          queue: 'site:nereval:js',
          maxInFlight: 4,
          rateLimit: { kind: 'token-bucket', ratePerSecond: 2, burst: 5 },
        },
      ],
    }),
    onClick: () => {},
  },
};

export const NoScripts: Story = {
  args: {
    site: createSiteDetail('minimal', {
      hasScripts: false,
      scriptCount: 0,
      scripts: [],
      verbCount: 1,
    }),
    onClick: () => {},
  },
};
