import { Box, LinearProgress, Typography } from '@mui/material';

interface TokenBucketGaugeProps {
  tokens: number;
  burst: number;
  ratePerSecond: number;
}

function gaugeColor(pct: number): 'success' | 'warning' | 'error' {
  if (pct > 50) return 'success';
  if (pct >= 25) return 'warning';
  return 'error';
}

export function TokenBucketGauge({ tokens, burst, ratePerSecond }: TokenBucketGaugeProps) {
  if (!burst) {
    return (
      <Typography variant="body2" color="text.secondary">
        No rate limiting
      </Typography>
    );
  }

  const pct = (tokens / burst) * 100;
  const color = gaugeColor(pct);
  const remaining = burst - tokens;
  const fullInSeconds = ratePerSecond > 0 ? remaining / ratePerSecond : Infinity;

  return (
    <Box>
      <LinearProgress
        variant="determinate"
        value={Math.min(Math.max(pct, 0), 100)}
        color={color}
        sx={{ height: 8, borderRadius: 1, mb: 0.5 }}
      />
      <Typography variant="caption" color="text.secondary">
        {tokens.toFixed(1)} / {burst.toFixed(1)} tokens
      </Typography>
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Typography variant="caption" color="text.secondary">
          Refills at {ratePerSecond.toFixed(1)}/sec
        </Typography>
        <Typography variant="caption" color="text.secondary">
          Full in ~{fullInSeconds.toFixed(1)}s
        </Typography>
      </Box>
    </Box>
  );
}
