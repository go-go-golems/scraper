import { Box, Button, Chip } from '@mui/material';
import type { ResultFilters } from './ResultFilterBar';

interface ActiveResultFilterChipsProps {
  filters: ResultFilters;
  onRemove: (field: keyof ResultFilters) => void;
  onClearAll: () => void;
}

export function ActiveResultFilterChips({ filters, onRemove, onClearAll }: ActiveResultFilterChipsProps) {
  const chips: { field: keyof ResultFilters; label: string }[] = [];

  if (filters.opId) {
    chips.push({ field: 'opId', label: `Op: ${filters.opId.split(':').pop()}` });
  }
  if (filters.kind) {
    chips.push({ field: 'kind', label: `Kind: ${filters.kind}` });
  }
  if (filters.status) {
    chips.push({ field: 'status', label: `Status: ${filters.status}` });
  }
  if (filters.search) {
    chips.push({ field: 'search', label: `Search: ${filters.search}` });
  }

  if (chips.length === 0) return null;

  return (
    <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.75, alignItems: 'center', mb: 1 }}>
      {chips.map(({ field, label }) => (
        <Chip
          key={field}
          label={label}
          size="small"
          onDelete={() => onRemove(field)}
          sx={{ fontFamily: 'monospace' }}
        />
      ))}
      <Button size="small" onClick={onClearAll}>
        Clear all
      </Button>
    </Box>
  );
}
