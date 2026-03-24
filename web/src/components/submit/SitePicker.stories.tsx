import type { Meta, StoryObj } from '@storybook/react';
import { SitePicker } from './SitePicker';
import { createSiteSummary } from '../../stories/__fixtures__/factories';

const meta: Meta<typeof SitePicker> = {
  title: 'Submit/SitePicker',
  component: SitePicker,
};

export default meta;
type Story = StoryObj<typeof SitePicker>;

const fourSites = [
  createSiteSummary('hackernews'),
  createSiteSummary('slashdot'),
  createSiteSummary('js-demo'),
  createSiteSummary('nereval'),
];

export const Default: Story = {
  args: {
    sites: fourSites,
    selected: null,
    onSelect: () => {},
  },
};

export const Selected: Story = {
  args: {
    sites: fourSites,
    selected: 'hackernews',
    onSelect: () => {},
  },
};

export const SingleSite: Story = {
  args: {
    sites: [createSiteSummary('hackernews')],
    selected: null,
    onSelect: () => {},
  },
};
