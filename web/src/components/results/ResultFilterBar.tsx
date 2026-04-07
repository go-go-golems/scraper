import { useState, useCallback, useRef } from 'react';
import {
  Box,
  Chip,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  TextField,
  Typography,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import type { WorkflowOp } from '../../api/types';

export interface ResultFilters {
  opId: string;
  kind: string;
  status: string;
  search: string;
}

interface ResultFilterBarProps {
  filters: ResultFilters;
  onFiltersChange: (filters: ResultFilters) => void;
  onSearchChange: (value: string) => void;
  searchInputValue: string;
  ops: WorkflowOp[];
}

const KIND_OPTIONS = [
  { value: '', label: 'All kinds' },
  { value: 'http', label: 'HTTP' },
  { value: 'js', label: 'JavaScript' },
  { value: 'http-response-body', label: 'HTTP response body' },
  { value: 'json-output', label: 'JSON output' },
  { value: 'exec-log', label: 'Execution log' },
  { value: 'page', label: 'Page' },
  { value: 'metadata', label: 'Metadata' },
];

const STATUS_OPTIONS = [
  { value: '', label: 'All statuses' },
  { value: 'succeeded', label: 'Succeeded' },
  { value: 'failed', label: 'Failed' },
  { value: 'running', label: 'Running' },
  { value: 'pending', label: 'Pending' },
  { value: 'canceled', label: 'Canceled' },
];

export function ResultFilterBar({
  filters,
  onFiltersChange,
  onSearchChange,
  searchInputValue,
  ops,
}: ResultFilterBarProps) {
  const handleChange = useCallback(
    (field: keyof ResultFilters) => (e: SelectChangeEvent<string>) => {
      onFiltersChange({ ...filters, [field]: e.target.value });
    },
    [filters, onFiltersChange],
  );

  const handleSearchKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        onFiltersChange({ ...filters, search: searchInputValue });
      }
    },
    [filters, onFiltersChange, searchInputValue],
  );

  return (
    <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap', alignItems: 'center', p: 1 }}>
      {/* Op filter */}
      <FormControl size="small" sx={{ minWidth: 160 }}>
        <InputLabel>Op</InputLabel>
        <Select
          value={filters.opId}
          label="Op"
          onChange={handleChange('opId')}
        >
          <MenuItem value=""><em>All Ops</em></MenuItem>
          {ops.map((op) => {
            const label = `${op.op.Kind}:${op.op.ID.split(':').pop()}`;
            return (
              <MenuItem key={op.op.ID} value={op.op.ID}>
                {label}
              </MenuItem>
            );
          })}
        </Select>
      </FormControl>

      {/* Kind filter */}
      <FormControl size="small" sx={{ minWidth: 140 }}>
        <InputLabel>Kind</InputLabel>
        <Select
          value={filters.kind}
          label="Kind"
          onChange={handleChange('kind')}
        >
          {KIND_OPTIONS.map((opt) => (
            <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Status filter */}
      <FormControl size="small" sx={{ minWidth: 140 }}>
        <InputLabel>Status</InputLabel>
        <Select
          value={filters.status}
          label="Status"
          onChange={handleChange('status')}
        >
          {STATUS_OPTIONS.map((opt) => (
            <MenuItem key={opt.value} value={opt.value}>{opt.label}</MenuItem>
          ))}
        </Select>
      </FormControl>

      {/* Search */}
      <TextField
        size="small"
        label="Search by name"
        value={searchInputValue}
        onChange={(e) => onSearchChange(e.target.value)}
        onKeyDown={handleSearchKeyDown}
        InputProps={{
          startAdornment: <SearchIcon sx={{ fontSize: 16, mr: 0.5, color: 'text.disabled' }} />,
        }}
        sx={{ flex: 1, minWidth: 160 }}
      />
    </Box>
  );
}
