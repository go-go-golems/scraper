import { Box, Button, Chip } from '@mui/material';
import type { ArtifactFilters } from './FilterBar';

interface ActiveFilterChipsProps {
  filters: ArtifactFilters;
  opNames: Record<string, string>; // opId → short display name
  onRemove: (field: keyof ArtifactFilters) => void;
  onClearAll: () => void;
}

export function ActiveFilterChips({ filters, opNames, onRemove, onClearAll }: ActiveFilterChipsProps) {
  const chips: { field: keyof ArtifactFilters; label: string }[] = [];

  if (filters.opId) {
    chips.push({ field: 'opId', label: `Op: ${opNames[filters.opId] ?? filters.opId}` });
  }
  if (filters.kind) {
    chips.push({ field: 'kind', label: `Kind: ${filters.kind}` });
  }
  if (filters.contentType) {
    chips.push({ field: 'contentType', label: `Type: ${filters.contentType}` });
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
