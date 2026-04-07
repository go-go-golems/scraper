import type { Meta, StoryObj } from '@storybook/react';
import { Box, Typography } from '@mui/material';
import type { ArtifactSummary } from '../../api/types';
import { ArtifactsPanel } from './ArtifactsPanel';

// NOTE: This is the Step 2 skeleton story.
// RTK Query provides data at runtime; these stories document the expected
// page states. Steps 3-6 will add full interactions.

const meta: Meta<typeof ArtifactsPanel> = {
  title: 'Artifacts/ArtifactsPanel',
  component: ArtifactsPanel,
  parameters: {
    layout: 'fullscreen',
  },
  tags: ['autodocs'],
};
export default meta;
type Story = StoryObj<typeof ArtifactsPanel>;

/** Expected layout at Step 2 skeleton. The artifacts table and filter bar are
 *  added in Steps 3-4. Preview panel in Step 5. */
export const Skeleton: Story = {
  args: {
    workflowId: 'hackernews-extract-frontpage-1775586649974859668',
  },
  render: (args) => (
    <Box sx={{ p: 2 }}>
      <Typography variant="caption" color="text.disabled" sx={{ mb: 1, display: 'block' }}>
        Step 2 — ArtifactsPanel skeleton. Filter bar, table, and preview panel
        added in Steps 3-5.
      </Typography>
      <ArtifactsPanel {...args} />
    </Box>
  ),
};

/** Document the empty state — shown when the workflow has produced no artifacts. */
export const EmptyState: Story = {
  name: 'Empty state',
  args: {
    workflowId: 'empty-workflow-000',
  },
  render: (args) => (
    <Box sx={{ p: 2 }}>
      <Typography variant="caption" color="text.disabled" sx={{ mb: 1, display: 'block' }}>
        Empty state — shown when workflow has no artifacts.
      </Typography>
      <ArtifactsPanel {...args} />
    </Box>
  ),
};
