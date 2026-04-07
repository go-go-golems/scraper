import { useState } from 'react';
import { Box, CircularProgress, Typography } from '@mui/material';
import { useGetWorkflowArtifactsQuery } from '../../api/workflowApi';

interface ArtifactsPanelProps {
  workflowId: string;
}

// NOTE: This is the Step 2 skeleton.
// Full filter bar → Step 3
// Artifact table with pagination → Step 4
// Preview panel → Step 5
// Bridge links → Step 6

export function ArtifactsPanel({ workflowId }: ArtifactsPanelProps) {
  const [page] = useState(0);
  const limit = 20;
  const offset = page * limit;

  const { data, isLoading, isError } = useGetWorkflowArtifactsQuery(
    { workflowId, limit, offset },
    { skip: !workflowId },
  );

  const artifacts = data?.artifacts ?? [];
  const total = data?.total ?? 0;

  if (isLoading) {
    return (
      <Box sx={{ p: 3, display: 'flex', justifyContent: 'center' }}>
        <CircularProgress size={24} />
      </Box>
    );
  }

  if (isError || !data) {
    return (
      <Box sx={{ p: 3 }}>
        <Typography color="error" variant="body2">
          Failed to load artifacts.
        </Typography>
      </Box>
    );
  }

  if (artifacts.length === 0) {
    return (
      <Box
        sx={{
          p: 4,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          gap: 1,
        }}
      >
        <Typography variant="h6" color="text.secondary">
          No artifacts yet
        </Typography>
        <Typography variant="body2" color="text.disabled">
          Artifacts will appear here once the workflow produces them.
        </Typography>
      </Box>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ color: 'text.caption', px: 1 }}>
        Showing {artifacts.length} of {total} artifacts
        {/* TODO Step 3: FilterBar */}
        {/* TODO Step 4: ArtifactTable with pagination */}
        {/* TODO Step 5: Preview panel */}
      </Box>
      <Box sx={{ px: 1 }}>
        <Typography variant="body2" color="text.secondary">
          {artifacts.length > 0
            ? `First artifact: ${artifacts[0].name} (${artifacts[0].kind})`
            : ''}
        </Typography>
      </Box>
    </Box>
  );
}
