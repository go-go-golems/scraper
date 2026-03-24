import { Card, CardContent, Typography, Box, Chip, Skeleton } from '@mui/material';

interface BreakdownItem {
  label: string;
  value: number;
  color: 'default' | 'primary' | 'secondary' | 'error' | 'info' | 'success' | 'warning';
}

interface StatCardProps {
  title: string;
  value: number;
  breakdown?: BreakdownItem[];
  loading?: boolean;
}

export function StatCard({ title, value, breakdown, loading }: StatCardProps) {
  if (loading) {
    return (
      <Card>
        <CardContent>
          <Skeleton width={80} height={20} />
          <Skeleton width={60} height={40} sx={{ mt: 1 }} />
          <Box sx={{ display: 'flex', gap: 0.5, mt: 1, flexWrap: 'wrap' }}>
            <Skeleton width={60} height={24} />
            <Skeleton width={60} height={24} />
          </Box>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </Typography>
        <Typography variant="h4" component="div">
          {value.toLocaleString()}
        </Typography>
        {breakdown && breakdown.length > 0 && (
          <Box sx={{ display: 'flex', gap: 0.5, mt: 1.5, flexWrap: 'wrap' }}>
            {breakdown.map((item) => (
              <Chip
                key={item.label}
                label={`${item.label}: ${item.value}`}
                size="small"
                color={item.color}
                variant="outlined"
              />
            ))}
          </Box>
        )}
      </CardContent>
    </Card>
  );
}
