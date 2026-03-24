import { AppBar, Box, Tab, Tabs, Toolbar, Typography } from '@mui/material';
import StorageIcon from '@mui/icons-material/Storage';
import { useLocation, useNavigate } from 'react-router-dom';

const tabRoutes = [
  { label: 'Overview', path: '/' },
  { label: 'Workflows', path: '/workflows' },
  { label: 'Queues', path: '/queues' },
  { label: 'Sites', path: '/sites' },
  { label: 'Submit', path: '/submit' },
] as const;

function currentTabIndex(pathname: string): number {
  if (pathname.startsWith('/workflows')) return 1;
  if (pathname.startsWith('/queues')) return 2;
  if (pathname.startsWith('/sites')) return 3;
  if (pathname.startsWith('/submit')) return 4;
  return 0;
}

interface AppShellProps {
  children: React.ReactNode;
}

export function AppShell({ children }: AppShellProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const tabIndex = currentTabIndex(location.pathname);

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: '100vh' }}>
      <AppBar position="static" elevation={0}>
        <Toolbar>
          <StorageIcon sx={{ mr: 1.5 }} />
          <Typography variant="h6" sx={{ flexGrow: 1 }}>
            Scraper Engine
          </Typography>
          <Tabs
            value={tabIndex}
            onChange={(_, value) => navigate(tabRoutes[value].path)}
            textColor="inherit"
            indicatorColor="secondary"
          >
            {tabRoutes.map((t) => (
              <Tab key={t.path} label={t.label} />
            ))}
          </Tabs>
        </Toolbar>
      </AppBar>
      <Box sx={{ flexGrow: 1, p: 3, bgcolor: 'grey.50' }}>
        {children}
      </Box>
    </Box>
  );
}
