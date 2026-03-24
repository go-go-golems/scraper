import { Box, Typography, Skeleton } from '@mui/material';

export interface LogEntry {
  timestamp: string;
  message: string;
}

interface OpExecutionLogProps {
  entries: LogEntry[];
  loading: boolean;
}

export function OpExecutionLog({ entries, loading }: OpExecutionLogProps) {
  if (loading) {
    return (
      <Box sx={{ p: 1.5, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {Array.from({ length: 6 }, (_, i) => (
          <Skeleton key={i} variant="text" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }} />
        ))}
      </Box>
    );
  }

  if (entries.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary" sx={{ p: 2 }}>
        No log output
      </Typography>
    );
  }

  return (
    <Box
      sx={{
        maxHeight: 400,
        overflow: 'auto',
        fontFamily: '"JetBrains Mono", "Fira Code", monospace',
        fontSize: '0.8rem',
        lineHeight: 1.6,
      }}
    >
      {entries.map((entry, i) => (
        <Box
          key={i}
          sx={{
            display: 'flex',
            gap: 2,
            px: 1.5,
            py: 0.25,
            bgcolor: i % 2 === 0 ? 'grey.50' : 'transparent',
          }}
        >
          <Box
            component="span"
            sx={{ color: 'text.secondary', whiteSpace: 'nowrap', flexShrink: 0 }}
          >
            {entry.timestamp}
          </Box>
          <Box component="span" sx={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {entry.message}
          </Box>
        </Box>
      ))}
    </Box>
  );
}
