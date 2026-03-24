import { ToggleButton, ToggleButtonGroup } from '@mui/material';
import type { SiteSummary } from '../../api/types';

interface SitePickerProps {
  sites: SiteSummary[];
  selected: string | null;
  onSelect: (site: string) => void;
}

export function SitePicker({ sites, selected, onSelect }: SitePickerProps) {
  return (
    <ToggleButtonGroup
      value={selected}
      exclusive
      onChange={(_event, value: string | null) => {
        if (value !== null) {
          onSelect(value);
        }
      }}
      size="small"
    >
      {sites.map((site) => (
        <ToggleButton key={site.name} value={site.name} sx={{ textTransform: 'none' }}>
          {site.name}
        </ToggleButton>
      ))}
    </ToggleButtonGroup>
  );
}
