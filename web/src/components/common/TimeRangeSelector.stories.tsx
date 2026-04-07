import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { TimeRangeSelector } from './TimeRangeSelector';
import type { TimeRange } from './TimeRangeSelector';
import { Typography } from '@mui/material';

function TimeRangeDemo({ initial }: { initial: TimeRange }) {
  const [value, setValue] = useState<TimeRange>(initial);
  return (
    <div style={{ width: 500 }}>
      <TimeRangeSelector value={value} onChange={setValue} />
      <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
        Current: mode={value.mode}
        {value.range ? `, range=${value.range}` : ''}
        {value.from ? `, from=${value.from}` : ''}
        {value.to ? `, to=${value.to}` : ''}
      </Typography>
    </div>
  );
}

const meta: Meta<typeof TimeRangeSelector> = {
  title: 'Common/TimeRangeSelector',
  component: TimeRangeSelector,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof TimeRangeSelector>;

export const LiveMode: Story = {
  render: () => <TimeRangeDemo initial={{ mode: 'live' }} />,
};

export const Relative6h: Story = {
  render: () => <TimeRangeDemo initial={{ mode: 'relative', range: '6h' }} />,
};

export const Relative24h: Story = {
  render: () => <TimeRangeDemo initial={{ mode: 'relative', range: '24h' }} />,
};

export const CustomMode: Story = {
  render: () => <TimeRangeDemo initial={{ mode: 'absolute' }} />,
};

export const MinimalOptions: Story = {
  render: () => (
    <TimeRangeDemo initial={{ mode: 'live' }} />
  ),
  args: {
    options: ['live', '1h', '24h'],
  },
};
