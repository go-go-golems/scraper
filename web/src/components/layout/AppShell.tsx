import { AppBar, Box, Tab, Tabs, Toolbar, Typography } from '@mui/material';
import StorageIcon from '@mui/icons-material/Storage';

export type TabId = 'overview' | 'workflows' | 'queues' | 'submit';

interface AppShellProps {
  currentTab: TabId;
  onTabChange: (tab: TabId) => void;
  children: React.ReactNode;
}

export function AppShell({ currentTab, onTabChange, children }: AppShellProps) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <AppBar position="static" elevation={0}>
        <Toolbar>
          <StorageIcon sx={{ mr: 1.5 }} />
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            Scraper Engine
          </Typography>
          <Tabs
            value={currentTab}
            onChange={(_, value) => onTabChange(value as TabId)}
            textColor="inherit"
            indicatorColor="secondary"
          >
            <Tab label="Overview" value="overview" />
            <Tab label="Workflows" value="workflows" />
            <Tab label="Queues" value="queues" />
            <Tab label="Submit" value="submit" />
          </Tabs>
        </Toolbar>
      </AppBar>
      <Box sx={{ flexGrow: 1, p: 3, bgcolor: 'grey.50' }}>
        {children}
      </Box>
    </Box>
  );
}
