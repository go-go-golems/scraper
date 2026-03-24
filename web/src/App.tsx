import { useCallback } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { Box, IconButton, Typography } from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { AppShell, type TabId } from './components/layout/AppShell';
import { EngineOverviewPage } from './pages/EngineOverviewPage';
import { WorkflowsPage } from './pages/WorkflowsPage';
import { WorkflowDetailPage } from './pages/WorkflowDetailPage';
import { QueueMonitorPage } from './pages/QueueMonitorPage';
import { SubmitWorkflowPage } from './pages/SubmitWorkflowPage';
import { setTab } from './store/uiSlice';
import type { RootState } from './store';
import { useState } from 'react';

function App() {
  const currentTab = useSelector((state: RootState) => state.ui.currentTab);
  const dispatch = useDispatch();
  const [selectedWorkflowId, setSelectedWorkflowId] = useState<string | null>(null);

  const handleTabChange = useCallback(
    (tab: TabId) => {
      dispatch(setTab(tab));
      setSelectedWorkflowId(null);
    },
    [dispatch],
  );

  const handleWorkflowClick = useCallback((id: string) => {
    setSelectedWorkflowId(id);
  }, []);

  const handleBackToList = useCallback(() => {
    setSelectedWorkflowId(null);
  }, []);

  let page;
  switch (currentTab) {
    case 'overview':
      page = <EngineOverviewPage />;
      break;
    case 'workflows':
      if (selectedWorkflowId) {
        page = (
          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 2 }}>
              <IconButton onClick={handleBackToList} size="small">
                <ArrowBackIcon />
              </IconButton>
              <Typography variant="body2" color="text.secondary">
                Back to Workflows
              </Typography>
            </Box>
            <WorkflowDetailPage workflowId={selectedWorkflowId} />
          </Box>
        );
      } else {
        page = <WorkflowsPage onWorkflowClick={handleWorkflowClick} />;
      }
      break;
    case 'queues':
      page = <QueueMonitorPage />;
      break;
    case 'submit':
      page = <SubmitWorkflowPage />;
      break;
  }

  return (
    <AppShell currentTab={currentTab} onTabChange={handleTabChange}>
      {page}
    </AppShell>
  );
}

export default App;
