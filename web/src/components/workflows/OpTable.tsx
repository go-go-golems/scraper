import {
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import CodeIcon from '@mui/icons-material/Code';
import HttpIcon from '@mui/icons-material/Http';
import type { WorkflowOp } from '../../api/types';
import { StatusChip } from '../common/StatusChip';

interface OpTableProps {
  ops: WorkflowOp[];
  selectedOpId: string | null;
  onSelectOp: (id: string) => void;
}

function KindIcon({ kind }: { kind: string }) {
  if (kind === 'js') return <CodeIcon fontSize="small" />;
  if (kind === 'http' || kind === 'fetch') return <HttpIcon fontSize="small" />;
  return <Typography variant="caption">{kind}</Typography>;
}

function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diffMs = now - then;
  const diffSec = Math.floor(diffMs / 1000);

  if (diffSec < 60) return `${diffSec}s ago`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  return `${diffDay}d ago`;
}

export function OpTable({ ops, selectedOpId, onSelectOp }: OpTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Kind</TableCell>
            <TableCell>Queue</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Retry</TableCell>
            <TableCell>Created</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {ops.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} align="center">
                <Typography variant="body2" color="text.disabled" sx={{ py: 4 }}>
                  No operations
                </Typography>
              </TableCell>
            </TableRow>
          )}
          {ops.map((wop) => (
            <TableRow
              key={wop.op.ID}
              hover
              selected={wop.op.ID === selectedOpId}
              sx={{ cursor: 'pointer' }}
              onClick={() => onSelectOp(wop.op.ID)}
            >
              <TableCell>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
                  noWrap
                >
                  {wop.op.ID}
                </Typography>
              </TableCell>
              <TableCell>
                <KindIcon kind={wop.op.Kind} />
              </TableCell>
              <TableCell>
                <Typography variant="caption" noWrap>
                  {wop.op.Queue}
                </Typography>
              </TableCell>
              <TableCell>
                <StatusChip status={wop.status} />
              </TableCell>
              <TableCell>
                <Typography variant="caption">
                  {wop.op.Retry.MaxAttempts > 1
                    ? `${wop.op.RetryState.Attempt}/${wop.op.Retry.MaxAttempts}`
                    : '-'}
                </Typography>
              </TableCell>
              <TableCell>
                <Typography variant="caption" color="text.secondary" noWrap>
                  {formatRelativeTime(wop.createdAt)}
                </Typography>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
