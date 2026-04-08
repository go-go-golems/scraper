import { Chip } from '@mui/material';

interface SiteOriginChipProps {
  originKind: string;
  size?: 'small' | 'medium';
}

export function SiteOriginChip({ originKind, size = 'small' }: SiteOriginChipProps) {
  if (originKind === 'manifest') {
    return <Chip label="Declarative" size={size} color="success" variant="outlined" />;
  }
  return <Chip label="Go Native" size={size} color="default" variant="outlined" />;
}
