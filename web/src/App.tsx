import { useCallback } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { AppShell, type TabId } from './components/layout/AppShell';
import { EngineOverviewPage } from './pages/EngineOverviewPage';
import { WorkflowsPage } from './pages/WorkflowsPage';
import { QueueMonitorPage } from './pages/QueueMonitorPage';
import { SubmitWorkflowPage } from './pages/SubmitWorkflowPage';
import { setTab } from './store/uiSlice';
import type { RootState } from './store';

function App() {
  const currentTab = useSelector((state: RootState) => state.ui.currentTab);
  const dispatch = useDispatch();

  const handleTabChange = useCallback(
    (tab: TabId) => dispatch(setTab(tab)),
    [dispatch],
  );

  let page;
  switch (currentTab) {
    case 'overview':
      page = <EngineOverviewPage />;
      break;
    case 'workflows':
      page = <WorkflowsPage />;
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
