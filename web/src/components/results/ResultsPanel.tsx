import { useState, useCallback, useRef } from 'react';
import {
  Box,
  CircularProgress,
  Divider,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material';
import ChevronLeftIcon from '@mui/icons-material/ChevronLeft';
import ChevronRightIcon from '@mui/icons-material/ChevronRight';
import PanelIcon from '@mui/icons-material/ViewAgenda';
import { useGetWorkflowResultsQuery, useGetWorkflowOpsQuery } from '../../api/workflowApi';
import { ResultFilterBar, type ResultFilters } from './ResultFilterBar';
import { ActiveResultFilterChips } from './ActiveResultFilterChips';
import { ResultsTable } from './ResultsTable';
import { ResultPreviewPanel } from './ResultPreviewPanel';

interface ResultsPanelProps {
  workflowId: string;
  /** When set (from OpResultTab bridge), pre-fill the Op filter */
  initialOpIdFilter?: string;
  /** Navigate to the Ops tab and open OpDetailDrawer for this op */
  onOpClick: (opId: string) => void;
}

const DEFAULT_FILTERS: ResultFilters = {
  opId: '',
  kind: '',
  status: '',
  search: '',
};

export function ResultsPanel({ workflowId, initialOpIdFilter, onOpClick }: ResultsPanelProps) {
  const [page, setPage] = useState(0);
  const [previewVisible, setPreviewVisible] = useState(true);
  const [selectedResultId, setSelectedResultId] = useState<string | null>(null);
  const [filters, setFilters] = useState<ResultFilters>({
    ...DEFAULT_FILTERS,
    opId: initialOpIdFilter ?? '',
  });
  const [searchInputValue, setSearchInputValue] = useState('');
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

  const { data, isLoading, isError } = useGetWorkflowResultsQuery(
    {
      workflowId,
      opId: filters.opId || undefined,
      kind: filters.kind || undefined,
      status: filters.status || undefined,
      search: filters.search || undefined,
      limit,
      offset,
    },
    { skip: !workflowId },
  );

  const results = data?.results ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / limit) || 1;
  const startItem = total === 0 ? 0 : offset + 1;
  const endItem = Math.min(offset + limit, total);

  const handleRemoveFilter = useCallback((field: keyof ResultFilters) => {
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
        <Typography color="error" variant="body2">Failed to load results.</Typography>
      </Box>
    );
  }

  const hasActiveFilters = [filters.opId, filters.kind, filters.status, filters.search].some(Boolean);

  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'row',
        minHeight: 500,
        gap: 1,
      }}
    >
      {/* ── Left panel: filter bar + table ──────────────────────────────── */}
      <Box
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          gap: 0.5,
          overflow: 'hidden',
        }}
      >
        <ResultFilterBar
          filters={filters}
          onFiltersChange={(next) => { setFilters(next); setPage(0); }}
          onSearchChange={handleSearchChange}
          searchInputValue={searchInputValue}
          ops={ops}
        />

        {hasActiveFilters && (
          <ActiveResultFilterChips
            filters={filters}
            onRemove={handleRemoveFilter}
            onClearAll={handleClearAll}
          />
        )}

        <Divider sx={{ my: 0.5 }} />

        {/* Summary + pagination */}
        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
          <Typography variant="body2" color="text.secondary">
            {total === 0
              ? 'No results'
              : `Showing ${startItem}–${endItem} of ${total} result${total === 1 ? '' : 's'}`}
          </Typography>

          <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
            <Tooltip title={previewVisible ? 'Hide preview panel' : 'Show preview panel'}>
              <IconButton
                size="small"
                onClick={() => setPreviewVisible((v) => !v)}
                color={previewVisible ? 'primary' : 'default'}
              >
                <PanelIcon fontSize="small" />
              </IconButton>
            </Tooltip>
            <Divider orientation="vertical" flexItem sx={{ mx: 0.5 }} />
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

        {/* Results table */}
        {results.length > 0 ? (
          <ResultsTable
            results={results}
            selectedId={selectedResultId}
            onSelectResult={(opId) => {
              setSelectedResultId(opId);
              if (!previewVisible) setPreviewVisible(true);
            }}
            onOpClick={onOpClick}
          />
        ) : !hasActiveFilters ? (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <Typography variant="h6" color="text.secondary">No results yet</Typography>
            <Typography variant="body2" color="text.disabled">
              Results will appear here once the workflow ops complete.
            </Typography>
          </Box>
        ) : (
          <Box sx={{ p: 4, textAlign: 'center' }}>
            <Typography variant="body2" color="text.secondary">
              No results match the current filters.
            </Typography>
          </Box>
        )}
      </Box>

      {/* ── Right panel: preview ─────────────────────────────────────── */}
      {previewVisible && (
        <Box
          sx={{
            flex: '0 0 45%',
            border: 1,
            borderColor: 'divider',
            borderRadius: 1,
            overflow: 'hidden',
            display: 'flex',
            flexDirection: 'column',
          }}
        >
          {selectedResultId ? (
            <ResultPreviewPanel
              result={data?.results.find((r) => r.opID === selectedResultId) ?? null}
              workflowId={workflowId}
              onClose={() => setSelectedResultId(null)}
              onOpClick={(opId) => {
                setSelectedResultId(null);
                onOpClick(opId);
              }}
            />
          ) : (
            <Box
              sx={{
                flex: 1,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}
            >
              <Typography variant="caption" color="text.disabled">
                Click a result row to preview
              </Typography>
            </Box>
          )}
        </Box>
      )}
    </Box>
  );
}
