import { Chip, type ChipProps } from '@mui/material';
import type { WorkflowStatus, OpStatus } from '../../api/types';

type StatusValue = WorkflowStatus | OpStatus;

const colorMap: Record<string, ChipProps['color']> = {
  pending: 'default',
  ready: 'info',
  running: 'info',
  succeeded: 'success',
  failed: 'error',
  canceled: 'warning',
};

interface StatusChipProps {
  status: StatusValue;
  size?: 'small' | 'medium';
}

export function StatusChip({ status, size = 'small' }: StatusChipProps) {
  return (
    <Chip
      label={status}
      color={colorMap[status] ?? 'default'}
      size={size}
      variant={status === 'running' ? 'filled' : 'outlined'}
    />
  );
}
