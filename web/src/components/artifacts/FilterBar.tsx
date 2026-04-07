import { useId } from 'react';
import {
  Box,
  FormControl,
  InputAdornment,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Typography,
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import type { SelectChangeEvent } from '@mui/material';
import type { WorkflowOp } from '../../api/types';

export interface ArtifactFilters {
  opId: string;
  kind: string;
  contentType: string;
  search: string;
}

interface FilterBarProps {
  filters: ArtifactFilters;
  onFiltersChange: (filters: ArtifactFilters) => void;
  onSearchChange: (value: string) => void;
  /** The live search input value — updated immediately as the user types,
   *  separate from the debounced filters.search value. */
  searchInputValue: string;
  ops: WorkflowOp[];
}

const KIND_OPTIONS = [
  { value: '', label: 'All kinds' },
  { value: 'http-response-body', label: 'HTTP response body' },
  { value: 'json-output', label: 'JSON output' },
  { value: 'exec-log', label: 'Execution log' },
  { value: 'extract', label: 'Extract' },
  { value: 'page', label: 'Page' },
  { value: 'screenshot', label: 'Screenshot' },
  { value: 'metadata', label: 'Metadata' },
  { value: 'raw', label: 'Raw / binary' },
];

const CONTENT_TYPE_OPTIONS = [
  { value: '', label: 'All types' },
  { value: 'text/html', label: 'text/html' },
  { value: 'application/json', label: 'application/json' },
  { value: 'text/plain', label: 'text/plain' },
  { value: 'image/png', label: 'image/png' },
  { value: 'image/jpeg', label: 'image/jpeg' },
  { value: 'application/octet-stream', label: 'binary / other' },
];

export function FilterBar({ filters, onFiltersChange, ops }: FilterBarProps) {
  const selectId = useId();

  const handleChange = (field: keyof ArtifactFilters) => (e: SelectChangeEvent<string> | React.ChangeEvent<HTMLInputElement>) => {
    onFiltersChange({ ...filters, [field]: e.target.value });
  };

  const activeFilterCount = [filters.opId, filters.kind, filters.contentType, filters.search].filter(Boolean).length;

  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1.5, alignItems: 'center', mb: 1 }}>
      {/* Op filter — populated from the workflow's ops list */}
      <FormControl size="small" sx={{ minWidth: 200 }}>
        <InputLabel id={`${selectId}-op-label`}>Op</InputLabel>
        <Select
          labelId={`${selectId}-op-label`}
          value={filters.opId}
          label="Op"
          onChange={handleChange('opId')}
        >
          <MenuItem value=""><em>All Ops</em></MenuItem>
          {ops.map((op) => (
            <MenuItem key={op.op.ID} value={op.op.ID}>
              <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                {op.op.Kind}:{op.op.ID.split(':').pop()}
              </Typography>
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Kind filter */}
      <FormControl size="small" sx={{ minWidth: 160 }}>
        <InputLabel id={`${selectId}-kind-label`}>Kind</InputLabel>
        <Select
          labelId={`${selectId}-kind-label`}
          value={filters.kind}
          label="Kind"
          onChange={handleChange('kind')}
        >
          {KIND_OPTIONS.map((o) => (
            <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Content type filter */}
      <FormControl size="small" sx={{ minWidth: 160 }}>
        <InputLabel id={`${selectId}-ct-label`}>Type</InputLabel>
        <Select
          labelId={`${selectId}-ct-label`}
          value={filters.contentType}
          label="Type"
          onChange={handleChange('contentType')}
        >
          {CONTENT_TYPE_OPTIONS.map((o) => (
            <MenuItem key={o.value} value={o.value}>{o.label}</MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Search — onSearchChange is debounced by the caller (ArtifactsPanel) */}
      <TextField
        size="small"
        label="Search by name"
        value={searchInputValue}
        onChange={(e) => onSearchChange(e.target.value)}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon fontSize="small" />
            </InputAdornment>
          ),
        }}
        sx={{ minWidth: 200 }}
      />

      {/* Active filter count badge */}
      {activeFilterCount > 0 && (
        <Typography variant="caption" color="text.secondary" sx={{ alignSelf: 'center' }}>
          {activeFilterCount} filter{activeFilterCount > 1 ? 's' : ''} active
        </Typography>
      )}
    </Box>
  );
}
