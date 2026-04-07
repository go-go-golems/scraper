import type { Meta, StoryObj } from '@storybook/react-vite';
import { useState } from 'react';
import { MultiSelectChipFilter } from './MultiSelectChipFilter';
import type { MultiSelectOption } from './MultiSelectChipFilter';
import { Box, Typography } from '@mui/material';

const severityOptions: MultiSelectOption[] = [
  { value: 'DEBUG', label: 'Debug', color: 'default' },
  { value: 'INFO', label: 'Info', color: 'info' },
  { value: 'WARN', label: 'Warn', color: 'warning' },
  { value: 'ERROR', label: 'Error', color: 'error' },
];

const sourceOptions: MultiSelectOption[] = [
  { value: 'SCHEDULER', label: 'Scheduler' },
  { value: 'WORKER', label: 'Worker' },
  { value: 'RUNNER', label: 'Runner' },
  { value: 'SERVER', label: 'Server' },
  { value: 'SUBMISSION', label: 'Submission' },
  { value: 'REQUEST', label: 'Request' },
];

function MultiSelectDemo({
  options,
  initial,
}: {
  options: MultiSelectOption[];
  initial: string[];
}) {
  const [selected, setSelected] = useState<string[]>(initial);
  return (
    <Box sx={{ width: 300 }}>
      <MultiSelectChipFilter
        label="Severity"
        options={options}
        selected={selected}
        onChange={setSelected}
      />
      <Typography variant="caption" sx={{ mt: 1, display: 'block' }}>
        Selected: {selected.length === 0 ? '(none — shows all)' : selected.join(', ')}
      </Typography>
    </Box>
  );
}

const meta: Meta<typeof MultiSelectChipFilter> = {
  title: 'Common/MultiSelectChipFilter',
  component: MultiSelectChipFilter,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof MultiSelectChipFilter>;

export const Empty: Story = {
  render: () => <MultiSelectDemo options={severityOptions} initial={[]} />,
};

export const MultipleSelected: Story = {
  render: () => (
    <MultiSelectDemo options={severityOptions} initial={['WARN', 'ERROR']} />
  ),
};

export const AllSelected: Story = {
  render: () => (
    <MultiSelectDemo
      options={severityOptions}
      initial={severityOptions.map((o) => o.value)}
    />
  ),
};

export const WithCustomColors: Story = {
  render: () => <MultiSelectDemo options={severityOptions} initial={['ERROR']} />,
};

export const Disabled: Story = {
  render: () => (
    <Box sx={{ width: 300 }}>
      <MultiSelectChipFilter
        label="Severity"
        options={severityOptions}
        selected={['WARN']}
        onChange={() => {}}
        disabled
      />
    </Box>
  ),
};

export const SourceFilter: Story = {
  render: () => (
    <MultiSelectDemo options={sourceOptions} initial={['WORKER', 'RUNNER']} />
  ),
};
