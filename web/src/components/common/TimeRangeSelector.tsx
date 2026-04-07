import { useState, useCallback } from 'react';
import { Box, Chip } from '@mui/material';
import { LocalizationProvider, DateRangePicker } from '@mui/x-date-pickers';
import { AdapterDayjs } from '@mui/x-date-pickers/AdapterDayjs';
import dayjs, { type Dayjs } from 'dayjs';

export type TimeRangeMode = 'live' | 'relative' | 'absolute';

export interface TimeRange {
  mode: TimeRangeMode;
  /** For 'relative' mode: e.g. '1h', '6h', '24h', '7d' */
  range?: string;
  /** For 'absolute' mode: ISO timestamp */
  from?: string;
  /** For 'absolute' mode: ISO timestamp */
  to?: string;
}

interface TimeRangeSelectorProps {
  value: TimeRange;
  onChange: (value: TimeRange) => void;
  options?: string[];
}

const DEFAULT_OPTIONS = ['live', '1h', '6h', '24h', '7d', 'custom'];

function chipLabel(opt: string): string {
  if (opt === 'live') return '● Live';
  if (opt === 'custom') return 'Custom';
  return `Last ${opt}`;
}

function isActive(mode: TimeRangeMode, opt: string, range?: string): boolean {
  if (opt === 'live') return mode === 'live';
  if (opt === 'custom') return mode === 'absolute';
  return mode === 'relative' && range === opt;
}

export function TimeRangeSelector({
  value,
  onChange,
  options = DEFAULT_OPTIONS,
}: TimeRangeSelectorProps) {
  const [customRange, setCustomRange] = useState<[Dayjs | null, Dayjs | null]>([
    dayjs().startOf('day'),
    dayjs(),
  ]);

  const handleSelect = useCallback(
    (opt: string) => {
      if (opt === 'live') {
        onChange({ mode: 'live' });
      } else if (opt === 'custom') {
        // Don't immediately fire — wait for picker confirmation
        onChange({ mode: 'absolute' });
      } else {
        onChange({ mode: 'relative', range: opt });
      }
    },
    [onChange],
  );

  const handleCustomChange = useCallback(
    (newValue: [Dayjs | null, Dayjs | null]) => {
      setCustomRange(newValue);
      const [from, to] = newValue;
      if (from && to) {
        onChange({
          mode: 'absolute',
          from: from.toISOString(),
          to: to.toISOString(),
        });
      }
    },
    [onChange],
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
      <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', alignItems: 'center' }}>
        {options.map((opt) => {
          const active = isActive(value.mode, opt, value.range);
          return (
            <Chip
              key={opt}
              label={chipLabel(opt)}
              variant={active ? 'filled' : 'outlined'}
              color={active ? 'primary' : 'default'}
              size="small"
              onClick={() => handleSelect(opt)}
              sx={{ cursor: 'pointer' }}
            />
          );
        })}
      </Box>

      {value.mode === 'absolute' && (
        <LocalizationProvider dateAdapter={AdapterDayjs}>
          <DateRangePicker
            value={customRange}
            onChange={handleCustomChange}
            slotProps={{
              textField: { size: 'small' },
            }}
          />
        </LocalizationProvider>
      )}
    </Box>
  );
}
