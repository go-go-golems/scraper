import { useState, useCallback, useRef } from 'react';
import { Box, CircularProgress, Divider, Typography } from '@mui/material';
import { useGetWorkflowArtifactsQuery, useGetWorkflowOpsQuery } from '../../api/workflowApi';
import type { WorkflowOp } from '../../api/types';
import { FilterBar, type ArtifactFilters } from './FilterBar';
import { ActiveFilterChips } from './ActiveFilterChips';

interface ArtifactsPanelProps {
  workflowId: string;
}

// NOTE: This is the Step 2-3 combined. Filter bar is Step 3.
// Full artifact table + pagination → Step 4
// Preview panel → Step 5
// Bridge links → Step 6

const DEFAULT_FILTERS: ArtifactFilters = {
  opId: '',
  kind: '',
  contentType: '',
  search: '',
};

function buildOpNameMap(ops: WorkflowOp[]): Record<string, string> {
  const result: Record<string, string> = {};
  for (const op of ops) {
    // Show "Kind:shortID" as the display name
    const shortId = op.op.ID.includes(':') ? op.op.ID.split(':').pop() : op.op.ID;
    result[op.op.ID] = `${op.op.Kind}:${shortId}`;
  }
  return result;
}

export function ArtifactsPanel({ workflowId }: ArtifactsPanelProps) {
  const [page, setPage] = useState(0);
  const [filters, setFilters] = useState<ArtifactFilters>(DEFAULT_FILTERS);
  const [searchInputValue, setSearchInputValue] = useState(''); // live, pre-debounce
  const searchTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  // Debounce: only apply search to filters after 300ms of no typing
  const handleSearchChange = useCallback((value: string) => {
    setSearchInputValue(value);
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => {
      setFilters((prev) => ({ ...prev, search: value }));
      setPage(0);
    }, 300);
  }, []);

  // Fetch ops for the dropdown
  const { data: ops = [] } = useGetWorkflowOpsQuery(workflowId, { skip: !workflowId });
  const opNameMap = buildOpNameMap(ops);

  const limit = 20;
  const offset = page * limit;

  const { data, isLoading, isError } = useGetWorkflowArtifactsQuery(
    {
      workflowId,
      opId: filters.opId || undefined,
      kind: filters.kind || undefined,
      contentType: filters.contentType || undefined,
      search: filters.search || undefined,
      limit,
      offset,
    },
    { skip: !workflowId },
  );

  const artifacts = data?.artifacts ?? [];
  const total = data?.total ?? 0;

  const handleRemoveFilter = useCallback((field: keyof ArtifactFilters) => {
    setFilters((prev) => {
      const next = { ...prev, [field]: '' };
      if (field === 'search') setSearchInputValue('');
      return next;
    });
    setPage(0);
  }, []);

  const handleClearAll = useCallback(() => {
    setFilters(DEFAULT_FILTERS);
    setSearchInputValue('');
    setPage(0);
  }, []);

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

  const hasActiveFilters = [filters.opId, filters.kind, filters.contentType, filters.search].some(Boolean);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      {/* Filter bar — search is debounced 300ms in the parent */}
      <FilterBar
        filters={filters}
        onFiltersChange={(next) => {
          setFilters(next);
          setPage(0);
        }}
        onSearchChange={handleSearchChange}
        searchInputValue={searchInputValue}
        ops={ops}
      />

      {/* Active filter chips */}
      {hasActiveFilters && (
        <ActiveFilterChips
          filters={filters}
          opNames={opNameMap}
          onRemove={handleRemoveFilter}
          onClearAll={handleClearAll}
        />
      )}

      <Divider sx={{ my: 0.5 }} />

      {/* Summary line */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Typography variant="body2" color="text.secondary">
          {total === 0
            ? 'No artifacts'
            : `Showing ${artifacts.length} of ${total} artifact${total === 1 ? '' : 's'}`}
        </Typography>
        {/* TODO Step 4: Pagination controls */}
      </Box>

      {/* Artifact list — TODO Step 4: replace with ArtifactTable */}
      {artifacts.length > 0 ? (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
          {artifacts.map((artifact) => (
            <Box
              key={artifact.id}
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1,
                px: 1,
                py: 0.5,
                borderRadius: 1,
                '&:hover': { bgcolor: 'action.hover' },
                cursor: 'default',
              }}
            >
              <Typography variant="caption" sx={{ color: 'text.disabled', minWidth: 24 }}>
                ◈
              </Typography>
              <Typography
                variant="body2"
                sx={{ fontFamily: 'monospace', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis' }}
                title={artifact.name}
              >
                {artifact.name}
              </Typography>
              <Typography variant="caption" color="text.disabled">
                {artifact.kind}
              </Typography>
              <Typography variant="caption" color="text.disabled">
                {artifact.size < 1024
                  ? `${artifact.size} B`
                  : artifact.size < 1024 * 1024
                  ? `${(artifact.size / 1024).toFixed(1)} KB`
                  : `${(artifact.size / 1024 / 1024).toFixed(1)} MB`}
              </Typography>
            </Box>
          ))}
        </Box>
      ) : !hasActiveFilters ? (
        <Box sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="h6" color="text.secondary">
            No artifacts yet
          </Typography>
          <Typography variant="body2" color="text.disabled">
            Artifacts will appear here once the workflow produces them.
          </Typography>
        </Box>
      ) : (
        <Box sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="body2" color="text.secondary">
            No artifacts match the current filters.
          </Typography>
        </Box>
      )}

      {/* TODO Step 5: Preview panel (right half of split pane) */}
    </Box>
  );
}
