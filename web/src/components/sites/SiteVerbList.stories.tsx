import type { Meta, StoryObj } from '@storybook/react';
import { SiteVerbList } from './SiteVerbList';
import { createVerbSummary } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof SiteVerbList> = {
  title: 'Sites/SiteVerbList',
  component: SiteVerbList,
};

export default meta;
type Story = StoryObj<typeof SiteVerbList>;

export const Default: Story = {
  args: {
    loading: false,
    verbs: [
      createVerbSummary({ name: 'seed', site: 'hackernews' }),
      createVerbSummary({ name: 'detail', site: 'hackernews' }),
    ],
  },
};

export const SingleVerb: Story = {
  args: {
    loading: false,
    verbs: [createVerbSummary({ name: 'seed', site: 'hackernews' })],
  },
};

export const Loading: Story = {
  args: {
    loading: true,
    verbs: [],
  },
};
