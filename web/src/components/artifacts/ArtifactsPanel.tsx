import { useState, useCallback, useRef } from 'react';
import { Box, CircularProgress, Divider, IconButton, Typography } from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import { useGetWorkflowArtifactsQuery, useGetWorkflowOpsQuery } from '../../api/workflowApi';
import type { WorkflowOp } from '../../api/types';
import { FilterBar, type ArtifactFilters } from './FilterBar';
import { ActiveFilterChips } from './ActiveFilterChips';
import { ArtifactTable } from './ArtifactTable';

interface ArtifactsPanelProps {
  workflowId: string;
}

// NOTE: Step 4 complete. ArtifactTable + pagination wired.
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
    const shortId = op.op.ID.includes(':') ? op.op.ID.split(':').pop() : op.op.ID;
    result[op.op.ID] = `${op.op.Kind}:${shortId}`;
  }
  return result;
}

export function ArtifactsPanel({ workflowId }: ArtifactsPanelProps) {
  const [page, setPage] = useState(0);
  const [selectedArtifactId, setSelectedArtifactId] = useState<string | null>(null); // TODO Step 5: wire to preview panel
  const [filters, setFilters] = useState<ArtifactFilters>(DEFAULT_FILTERS);
  const [searchInputValue, setSearchInputValue] = useState(''); // live, pre-debounce
  const searchTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  const handleSearchChange = useCallback((value: string) => {
    setSearchInputValue(value);
    if (searchTimer.current) clearTimeout(searchTimer.current);
    searchTimer.current = setTimeout(() => {
      setFilters((prev) => ({ ...prev, search: value }));
      setPage(0);
    }, 300);
  }, []);

  const limit = 20;
  const offset = page * limit;

  const { data: ops = [] } = useGetWorkflowOpsQuery(workflowId, { skip: !workflowId });
  const opNameMap = buildOpNameMap(ops);

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
  const totalPages = Math.ceil(total / limit) || 1;
  const startItem = total === 0 ? 0 : offset + 1;
  const endItem = Math.min(offset + limit, total);

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
        <Typography color="error" variant="body2">Failed to load artifacts.</Typography>
      </Box>
    );
  }

  const hasActiveFilters = [filters.opId, filters.kind, filters.contentType, filters.search].some(Boolean);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <FilterBar
        filters={filters}
        onFiltersChange={(next) => { setFilters(next); setPage(0); }}
        onSearchChange={handleSearchChange}
        searchInputValue={searchInputValue}
        ops={ops}
      />

      {hasActiveFilters && (
        <ActiveFilterChips
          filters={filters}
          opNames={opNameMap}
          onRemove={handleRemoveFilter}
          onClearAll={handleClearAll}
        />
      )}

      <Divider sx={{ my: 0.5 }} />

      {/* Summary + pagination */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
        <Typography variant="body2" color="text.secondary">
          {total === 0
            ? 'No artifacts'
            : `Showing ${startItem}–${endItem} of ${total} artifact${total === 1 ? '' : 's'}`}
        </Typography>

        <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
          <IconButton
            size="small"
            onClick={() => setPage((p) => Math.max(0, p - 1))}
            disabled={page === 0}
          >
            <ChevronLeftIcon fontSize="small" />
          </IconButton>
          <Typography variant="caption" color="text.secondary">
            Page {page + 1} of {totalPages}
          </Typography>
          <IconButton
            size="small"
            onClick={() => setPage((p) => Math.min(totalPages - 1, p + 1))}
            disabled={page >= totalPages - 1}
          >
            <ChevronRightIcon fontSize="small" />
          </IconButton>
        </Box>
      </Box>

      {/* Artifact table */}
      {artifacts.length > 0 ? (
        <ArtifactTable
          artifacts={artifacts}
          selectedId={selectedArtifactId}
          onSelectArtifact={setSelectedArtifactId}
          opNameMap={opNameMap}
        />
      ) : !hasActiveFilters ? (
        <Box sx={{ p: 4, textAlign: 'center' }}>
          <Typography variant="h6" color="text.secondary">No artifacts yet</Typography>
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

      {/* TODO Step 5: ArtifactPreviewPanel — right half of split pane */}
    </Box>
  );
}
