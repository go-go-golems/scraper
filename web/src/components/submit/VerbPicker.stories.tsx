import type { Meta, StoryObj } from '@storybook/react';
import { VerbPicker } from './VerbPicker';
import { createVerbSummary } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof VerbPicker> = {
  title: 'Submit/VerbPicker',
  component: VerbPicker,
};

export default meta;
type Story = StoryObj<typeof VerbPicker>;

const twoVerbs = [
  createVerbSummary({ name: 'seed', site: 'hackernews' }),
  createVerbSummary({ name: 'crawl', site: 'hackernews' }),
];

export const Default: Story = {
  args: {
    verbs: twoVerbs,
    selected: null,
    onSelect: () => {},
    loading: false,
  },
};

export const SingleVerb: Story = {
  args: {
    verbs: [createVerbSummary({ name: 'seed', site: 'hackernews' })],
    selected: null,
    onSelect: () => {},
    loading: false,
  },
};

export const Loading: Story = {
  args: {
    verbs: [],
    selected: null,
    onSelect: () => {},
    loading: true,
  },
};
