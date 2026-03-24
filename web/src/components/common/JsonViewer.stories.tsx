import type { Meta, StoryObj } from '@storybook/react';
import { JsonViewer } from './JsonViewer';

const meta: Meta<typeof JsonViewer> = {
  title: 'Common/JsonViewer',
  component: JsonViewer,
};

export default meta;
type Story = StoryObj<typeof JsonViewer>;

export const SimpleObject: Story = {
  args: {
    data: {
      baseURL: 'https://news.ycombinator.com/',
      maxPages: 2,
      category: 'top',
    },
  },
};

export const Nested: Story = {
  args: {
    data: {
      workflow: {
        id: 'wf-001',
        site: 'hackernews',
        config: {
          retries: 3,
          timeout: 30000,
          headers: {
            'User-Agent': 'scraper/1.0',
            Accept: 'text/html',
          },
        },
      },
      results: [
        { url: 'https://example.com/1', status: 200 },
        { url: 'https://example.com/2', status: 404 },
      ],
    },
  },
};

export const LargeArray: Story = {
  args: {
    data: Array.from({ length: 50 }, (_, i) => ({
      id: i + 1,
      title: `Item ${i + 1}`,
      score: Math.floor(Math.random() * 1000),
    })),
    maxHeight: 400,
  },
};

export const NullValue: Story = {
  args: {
    data: null,
  },
};
