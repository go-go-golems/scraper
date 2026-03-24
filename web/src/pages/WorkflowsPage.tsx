import { useCallback } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { Box, Card, CardContent, Typography } from '@mui/material';
import { WorkflowFilters } from '../components/workflows/WorkflowFilters';
import { WorkflowTable } from '../components/workflows/WorkflowTable';
import { useListWorkflowsQuery } from '../api/workflowApi';
import { setWorkflowFilters } from '../store/uiSlice';
import type { RootState } from '../store';

const sites = ['hackernews', 'slashdot', 'js-demo', 'nereval'];

interface WorkflowsPageProps {
  onWorkflowClick: (id: string) => void;
}

export function WorkflowsPage({ onWorkflowClick }: WorkflowsPageProps) {
  const dispatch = useDispatch();
  const { site, status } = useSelector((state: RootState) => state.ui.workflowFilters);

  const { data, isLoading } = useListWorkflowsQuery(
    {
      site: site || undefined,
      status: status || undefined,
      limit: 50,
    },
    { pollingInterval: 5000 },
  );

  const handleSiteChange = useCallback(
    (s: string) => dispatch(setWorkflowFilters({ site: s })),
    [dispatch],
  );

  const handleStatusChange = useCallback(
    (s: string) => dispatch(setWorkflowFilters({ status: s })),
    [dispatch],
  );

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Typography variant="h6">Workflows</Typography>

      <WorkflowFilters
        sites={sites}
        selectedSite={site}
        selectedStatus={status}
        onSiteChange={handleSiteChange}
        onStatusChange={handleStatusChange}
      />

      <Card>
        <CardContent sx={{ p: 0, '&:last-child': { pb: 0 } }}>
          <WorkflowTable
            workflows={data?.workflows ?? []}
            loading={isLoading}
            onWorkflowClick={onWorkflowClick}
          />
        </CardContent>
      </Card>
    </Box>
  );
}
