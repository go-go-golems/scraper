import {
  Box,
  Chip,
  LinearProgress,
  Link,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import type { WorkflowListItem } from '../../api/types';
import { StatusChip } from '../common/StatusChip';

interface WorkflowTableProps {
  workflows: WorkflowListItem[];
  loading: boolean;
  onWorkflowClick: (id: string) => void;
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

function SkeletonRows() {
  return (
    <>
      {Array.from({ length: 5 }).map((_, i) => (
        <TableRow key={i}>
          <TableCell><Skeleton width={100} /></TableCell>
          <TableCell><Skeleton width={80} /></TableCell>
          <TableCell><Skeleton width={120} /></TableCell>
          <TableCell><Skeleton width={70} /></TableCell>
          <TableCell><Skeleton width={140} /></TableCell>
          <TableCell><Skeleton width={60} /></TableCell>
        </TableRow>
      ))}
    </>
  );
}

export function WorkflowTable({ workflows, loading, onWorkflowClick }: WorkflowTableProps) {
  return (
    <TableContainer>
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>ID</TableCell>
            <TableCell>Site</TableCell>
            <TableCell>Name</TableCell>
            <TableCell>Status</TableCell>
            <TableCell>Progress</TableCell>
            <TableCell>Created</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {loading && <SkeletonRows />}
          {!loading && workflows.length === 0 && (
            <TableRow>
              <TableCell colSpan={6} align="center">
                <Typography variant="body2" color="text.disabled" sx={{ py: 4 }}>
                  No workflows found
                </Typography>
              </TableCell>
            </TableRow>
          )}
          {!loading &&
            workflows.map((item) => {
              const { workflow, opTotal, opDone } = item;
              const pct = opTotal > 0 ? (opDone / opTotal) * 100 : 0;
              return (
                <TableRow
                  key={workflow.ID}
                  hover
                  sx={{ cursor: 'pointer' }}
                  onClick={() => onWorkflowClick(workflow.ID)}
                >
                  <TableCell>
                    <Link
                      component="button"
                      variant="body2"
                      underline="hover"
                      onClick={(e) => {
                        e.stopPropagation();
                        onWorkflowClick(workflow.ID);
                      }}
                      sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
                    >
                      {workflow.ID}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Chip label={workflow.Site} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>{workflow.Name}</TableCell>
                  <TableCell>
                    <StatusChip status={workflow.Status} />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, minWidth: 140 }}>
                      <LinearProgress
                        variant="determinate"
                        value={pct}
                        sx={{ flexGrow: 1, height: 6, borderRadius: 1 }}
                      />
                      <Typography variant="caption" color="text.secondary" noWrap>
                        {opDone}/{opTotal}
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Typography variant="caption" color="text.secondary" noWrap>
                      {formatRelativeTime(workflow.CreatedAt)}
                    </Typography>
                  </TableCell>
                </TableRow>
              );
            })}
        </TableBody>
      </Table>
    </TableContainer>
  );
}
