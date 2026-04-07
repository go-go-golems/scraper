import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Box } from '@mui/material';
import { ActiveFilterChips } from './ActiveFilterChips';
import type { ArtifactFilters } from './FilterBar';

const meta: Meta<typeof ActiveFilterChips> = {
  title: 'Artifacts/ActiveFilterChips',
  component: ActiveFilterChips,
};
export default meta;
type Story = StoryObj<typeof ActiveFilterChips>;

const OP_NAME_MAP = {
  'wf-1:frontpage-extract': 'js:frontpage-extract',
  'wf-1:frontpage-fetch': 'http:frontpage-fetch',
};

export const MultipleFilters: Story = {
  render: () => {
    const [filters, setFilters] = useState<ArtifactFilters>({
      opId: 'wf-1:frontpage-extract',
      kind: 'json-output',
      contentType: 'application/json',
      search: 'summary',
    });
    return (
      <Box sx={{ p: 2 }}>
        <ActiveFilterChips
          filters={filters}
          opNames={OP_NAME_MAP}
          onRemove={(field) => setFilters((prev) => ({ ...prev, [field]: '' }))}
          onClearAll={() => setFilters({ opId: '', kind: '', contentType: '', search: '' })}
        />
      </Box>
    );
  },
};

export const SingleFilter: Story = {
  name: 'Single filter',
  render: () => {
    const [filters, setFilters] = useState<ArtifactFilters>({
      opId: '', kind: 'json-output', contentType: '', search: '',
    });
    return (
      <Box sx={{ p: 2 }}>
        <ActiveFilterChips
          filters={filters}
          opNames={OP_NAME_MAP}
          onRemove={(field) => setFilters((prev) => ({ ...prev, [field]: '' }))}
          onClearAll={() => setFilters({ opId: '', kind: '', contentType: '', search: '' })}
        />
      </Box>
    );
  },
};
