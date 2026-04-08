import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Box } from '@mui/material';
import { ResultFilterBar, type ResultFilters } from './ResultFilterBar';
import { defaultResultsHandlers, STORY_WORKFLOW_ID } from '../../stories/msw/handlers';
import type { WorkflowOp } from '../../api/types';

const meta: Meta<typeof ResultFilterBar> = {
  title: 'Results/ResultFilterBar',
  component: ResultFilterBar,
  parameters: {
    msw: { handlers: defaultResultsHandlers },
  },
};
export default meta;
type Story = StoryObj<typeof ResultFilterBar>;

const DEFAULT_FILTERS: ResultFilters = { opId: '', kind: '', status: '', search: '' };

export const Default: Story = {
  render: () => {
    const [filters, setFilters] = useState<ResultFilters>(DEFAULT_FILTERS);
    return (
      <Box sx={{ p: 2, maxWidth: 700 }}>
        <ResultFilterBar
          filters={filters}
          onFiltersChange={setFilters}
          onSearchChange={() => {}}
          searchInputValue=""
          ops={[]}
        />
      </Box>
    );
  },
};

export const WithActiveFilters: Story = {
  name: 'With active filters',
  render: () => {
    const [filters, setFilters] = useState<ResultFilters>({
      opId: `${STORY_WORKFLOW_ID}:extract`,
      kind: 'js',
      status: 'succeeded',
      search: 'extract',
    });
    return (
      <Box sx={{ p: 2, maxWidth: 700 }}>
        <ResultFilterBar
          filters={filters}
          onFiltersChange={setFilters}
          onSearchChange={() => {}}
          searchInputValue="extract"
          ops={[]}
        />
      </Box>
    );
  },
};
