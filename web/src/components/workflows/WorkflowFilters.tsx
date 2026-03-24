import {
  Box,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material';

interface WorkflowFiltersProps {
  sites: string[];
  selectedSite: string;
  selectedStatus: string;
  searchText?: string;
  onSiteChange: (site: string) => void;
  onStatusChange: (status: string) => void;
  onSearchChange?: (text: string) => void;
}

const statusOptions = ['all', 'pending', 'running', 'succeeded', 'failed', 'canceled'];

export function WorkflowFilters({
  sites,
  selectedSite,
  selectedStatus,
  searchText = '',
  onSiteChange,
  onStatusChange,
  onSearchChange,
}: WorkflowFiltersProps) {
  const siteOptions = ['all', ...sites];

  return (
    <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
      <FormControl size="small" sx={{ minWidth: 160 }}>
        <InputLabel id="site-filter-label">Site</InputLabel>
        <Select
          labelId="site-filter-label"
          value={selectedSite || 'all'}
          label="Site"
          onChange={(e: SelectChangeEvent) =>
            onSiteChange(e.target.value === 'all' ? '' : e.target.value)
          }
        >
          {siteOptions.map((s) => (
            <MenuItem key={s} value={s}>
              {s}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      <FormControl size="small" sx={{ minWidth: 160 }}>
        <InputLabel id="status-filter-label">Status</InputLabel>
        <Select
          labelId="status-filter-label"
          value={selectedStatus || 'all'}
          label="Status"
          onChange={(e: SelectChangeEvent) =>
            onStatusChange(e.target.value === 'all' ? '' : e.target.value)
          }
        >
          {statusOptions.map((s) => (
            <MenuItem key={s} value={s}>
              {s}
            </MenuItem>
          ))}
        </Select>
      </FormControl>

      <TextField
        size="small"
        label="Search"
        placeholder="Filter by name or ID..."
        value={searchText}
        onChange={(e) => onSearchChange?.(e.target.value)}
        sx={{ minWidth: 200 }}
      />
    </Box>
  );
}
