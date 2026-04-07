import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Box } from '@mui/material';
import { FilterBar, type ArtifactFilters } from './FilterBar';
import type { WorkflowOp } from '../../api/types';

const meta: Meta<typeof FilterBar> = {
  title: 'Artifacts/FilterBar',
  component: FilterBar,
};
export default meta;
type Story = StoryObj<typeof FilterBar>;

const FAKE_OPS: WorkflowOp[] = [
  {
    op: { ID: 'wf-1:frontpage-fetch', WorkflowID: 'wf-1', Site: 'hackernews', Kind: 'http', Queue: 'q', DedupKey: 'k', Input: {}, DependsOn: [], Retry: { MaxAttempts: 1, BackoffKind: '', InitialBackoff: 0, MaxBackoff: 0, Multiplier: 0 }, RetryState: { Attempt: 1, LastError: '' }, Metadata: {} },
    status: 'succeeded',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
  {
    op: { ID: 'wf-1:frontpage-extract', WorkflowID: 'wf-1', Site: 'hackernews', Kind: 'js', Queue: 'q', DedupKey: 'k', Input: {}, DependsOn: [], Retry: { MaxAttempts: 3, BackoffKind: 'exp', InitialBackoff: 1, MaxBackoff: 60, Multiplier: 2 }, RetryState: { Attempt: 1, LastError: '' }, Metadata: { script: 'extract.js' } },
    status: 'succeeded',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
  },
];

const DEFAULT_FILTERS: ArtifactFilters = { opId: '', kind: '', contentType: '', search: '' };

export const Default: Story = {
  render: () => {
    const [filters, setFilters] = useState<ArtifactFilters>(DEFAULT_FILTERS);
    return (
      <Box sx={{ p: 2, maxWidth: 700 }}>
        <FilterBar
          filters={filters}
          onFiltersChange={setFilters}
          onSearchChange={() => {}}
          searchInputValue={''}
          ops={FAKE_OPS}
        />
      </Box>
    );
  },
};

export const WithActiveFilters: Story = {
  name: 'With active filters',
  render: () => {
    const [filters, setFilters] = useState<ArtifactFilters>({
      opId: 'wf-1:frontpage-extract',
      kind: 'json-output',
      contentType: 'application/json',
      search: 'summary',
    });
    return (
      <Box sx={{ p: 2, maxWidth: 700 }}>
        <FilterBar
          filters={filters}
          onFiltersChange={setFilters}
          onSearchChange={() => {}}
          searchInputValue="summary"
          ops={FAKE_OPS}
        />
      </Box>
    );
  },
};
