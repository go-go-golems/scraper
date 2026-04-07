import { Box, Typography } from '@mui/material';

type SeverityLevel = number;

const severityColors: Record<string, string> = {
  DEBUG: '#9e9e9e',
  INFO: '#1976d2',
  WARN: '#ed6c02',
  ERROR: '#d32f2f',
};

function severityLabel(level: SeverityLevel): string {
  const map: Record<number, string> = {
    1: 'DEBUG',
    2: 'INFO',
    3: 'WARN',
    4: 'ERROR',
  };
  return map[level] ?? 'UNKNOWN';
}

interface SeverityDotIndicatorProps {
  severity: SeverityLevel;
}

export function SeverityDotIndicator({ severity }: SeverityDotIndicatorProps) {
  const label = severityLabel(severity);
  const color = severityColors[label] ?? '#9e9e9e';

  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.75 }}>
      <Box
        sx={{
          width: 10,
          height: 10,
          borderRadius: '50%',
          bgcolor: color,
          flexShrink: 0,
        }}
      />
      <Typography variant="caption" sx={{ fontWeight: 500 }}>
        {label}
      </Typography>
    </Box>
  );
}
