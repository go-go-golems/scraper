import { Box, Typography, useTheme } from '@mui/material';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

export interface ThroughputPoint {
  time: string;
  opsPerMin: number;
}

export interface ThroughputSeries {
  queueKey: string;
  points: ThroughputPoint[];
}

interface ThroughputChartProps {
  data: ThroughputSeries[];
  timeRange: '5m' | '15m' | '1h';
}

const COLORS = ['#1976d2', '#2e7d32', '#ed6c02', '#9c27b0', '#d32f2f', '#0288d1'];

function buildChartData(data: ThroughputSeries[]) {
  if (data.length === 0) return [];

  // Use the first series' time points as the x-axis
  const timePoints = data[0].points.map((p) => p.time);
  return timePoints.map((time, i) => {
    const row: Record<string, string | number> = { time };
    for (const series of data) {
      row[series.queueKey] = series.points[i]?.opsPerMin ?? 0;
    }
    return row;
  });
}

export function ThroughputChart({ data, timeRange }: ThroughputChartProps) {
  const theme = useTheme();
  const chartData = buildChartData(data);

  if (data.length === 0) {
    return (
      <Box sx={{ py: 4, textAlign: 'center' }}>
        <Typography variant="body2" color="text.disabled">
          No throughput data available
        </Typography>
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
        <Typography variant="body2" color="text.secondary">
          Ops completed per minute
        </Typography>
        <Typography variant="caption" color="text.disabled">
          Last {timeRange}
        </Typography>
      </Box>
      <ResponsiveContainer width="100%" height={300}>
        <LineChart data={chartData} margin={{ top: 5, right: 20, left: 0, bottom: 5 }}>
          <CartesianGrid strokeDasharray="3 3" stroke={theme.palette.divider} />
          <XAxis
            dataKey="time"
            tick={{ fontSize: 11 }}
            stroke={theme.palette.text.secondary}
          />
          <YAxis
            tick={{ fontSize: 11 }}
            stroke={theme.palette.text.secondary}
            allowDecimals={false}
          />
          <Tooltip
            contentStyle={{
              backgroundColor: theme.palette.background.paper,
              border: `1px solid ${theme.palette.divider}`,
              borderRadius: 4,
              fontSize: 12,
            }}
          />
          <Legend wrapperStyle={{ fontSize: 12 }} />
          {data.map((series, i) => (
            <Line
              key={series.queueKey}
              type="monotone"
              dataKey={series.queueKey}
              stroke={COLORS[i % COLORS.length]}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </Box>
  );
}
