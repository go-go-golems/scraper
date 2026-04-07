import type { Meta, StoryObj } from '@storybook/react';
import { OpInputTab } from './OpInputTab';

const meta: Meta<typeof OpInputTab> = {
  title: 'Workflows/OpDetail/OpInputTab',
  component: OpInputTab,
};

export default meta;
type Story = StoryObj<typeof OpInputTab>;

export const Default: Story = {
  args: {
    input: { baseURL: 'https://news.ycombinator.com/', maxPages: 2 },
  },
};

export const NullInput: Story = {
  args: {
    input: null,
  },
};

export const NestedObject: Story = {
  args: {
    input: {
      request: {
        url: 'https://news.ycombinator.com/',
        method: 'GET',
        headers: { 'User-Agent': 'scraper/1.0' },
      },
      timeout: 30000,
      retries: 3,
    },
  },
};

export const ArrayInput: Story = {
  args: {
    input: ['https://example.com/page1', 'https://example.com/page2', 'https://example.com/page3'],
  },
};
