import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AppErrorBoundary } from './components/common/AppErrorBoundary';
import { ToastProvider } from './components/common/ToastProvider';
import { AppShell } from './components/layout/AppShell';
import { EngineOverviewPage } from './pages/EngineOverviewPage';
import { WorkflowsPage } from './pages/WorkflowsPage';
import { WorkflowDetailPage } from './pages/WorkflowDetailPage';
import { QueueMonitorPage } from './pages/QueueMonitorPage';
import { SitesListPage } from './pages/SitesListPage';
import { SiteDetailPage } from './pages/SiteDetailPage';
import { SubmitWorkflowPage } from './pages/SubmitWorkflowPage';
import { RuntimeEventsPage } from './pages/RuntimeEventsPage';

function App() {
  return (
    <BrowserRouter>
      <AppErrorBoundary>
        <ToastProvider>
          <AppShell>
        <Routes>
          <Route path="/" element={<EngineOverviewPage />} />
          <Route path="/workflows" element={<WorkflowsPage />} />
          <Route path="/workflows/:workflowId" element={<WorkflowDetailPage />} />
          <Route path="/queues" element={<QueueMonitorPage />} />
          <Route path="/events" element={<RuntimeEventsPage />} />
          <Route path="/sites" element={<SitesListPage />} />
          <Route path="/sites/:siteName" element={<SiteDetailPage />} />
          <Route path="/submit" element={<SubmitWorkflowPage />} />
        </Routes>
          </AppShell>
        </ToastProvider>
      </AppErrorBoundary>
    </BrowserRouter>
  );
}

export default App;
