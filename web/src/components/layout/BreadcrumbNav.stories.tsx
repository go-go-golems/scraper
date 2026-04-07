import type { Meta, StoryObj } from '@storybook/react-vite';
import { MemoryRouter, useLocation } from 'react-router-dom';
import { BreadcrumbNav } from './BreadcrumbNav';

function LocationDisplay() {
  const location = useLocation();
  return (
    <div style={{ fontSize: 12, color: '#666', marginTop: 8 }}>
      Current path: {location.pathname}
    </div>
  );
}

const meta: Meta<typeof BreadcrumbNav> = {
  title: 'Layout/BreadcrumbNav',
  component: BreadcrumbNav,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof BreadcrumbNav>;

export const TopLevel: Story = {
  render: () => (
    <MemoryRouter initialEntries={['/']}>
      <BreadcrumbNav />
      <LocationDisplay />
    </MemoryRouter>
  ),
};

export const WorkflowsList: Story = {
  render: () => (
    <MemoryRouter initialEntries={['/workflows']}>
      <BreadcrumbNav />
      <LocationDisplay />
    </MemoryRouter>
  ),
};

export const WorkflowDetail: Story = {
  render: () => (
    <MemoryRouter
      initialEntries={[
        { pathname: '/workflows/wf-abc123', state: { workflowName: 'scrape-hackernews' } },
      ]}
    >
      <BreadcrumbNav />
      <LocationDisplay />
    </MemoryRouter>
  ),
};

export const SiteDetail: Story = {
  render: () => (
    <MemoryRouter initialEntries={['/sites/hackernews']}>
      <BreadcrumbNav />
      <LocationDisplay />
    </MemoryRouter>
  ),
};
