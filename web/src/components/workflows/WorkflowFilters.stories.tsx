import type { Meta, StoryObj } from '@storybook/react';
import { WorkflowFilters } from './WorkflowFilters';

const meta: Meta<typeof WorkflowFilters> = {
  title: 'Workflows/WorkflowFilters',
  component: WorkflowFilters,
};

export default meta;
type Story = StoryObj<typeof WorkflowFilters>;

const defaultSites = ['hackernews', 'slashdot', 'js-demo', 'nereval'];

export const Default: Story = {
  args: {
    sites: defaultSites,
    selectedSite: '',
    selectedStatus: '',
    searchText: '',
    onSiteChange: () => {},
    onStatusChange: () => {},
    onSearchChange: () => {},
  },
};

export const Filtered: Story = {
  args: {
    sites: defaultSites,
    selectedSite: 'hackernews',
    selectedStatus: 'running',
    searchText: '',
    onSiteChange: () => {},
    onStatusChange: () => {},
    onSearchChange: () => {},
  },
};
