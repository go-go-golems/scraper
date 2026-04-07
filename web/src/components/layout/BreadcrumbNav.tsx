import { useLocation, useNavigate } from 'react-router-dom';
import { Box, Breadcrumbs, Link, Typography } from '@mui/material';
import NavigateNextIcon from '@mui/icons-material/NavigateNext';

interface Crumb {
  label: string;
  path?: string;
}

function deriveCrumbs(pathname: string, state: Record<string, unknown> | null): Crumb[] {
  const crumbs: Crumb[] = [];

  if (pathname === '/') {
    return [{ label: 'Overview' }];
  }

  if (pathname.startsWith('/workflows/')) {
    crumbs.push({ label: 'Workflows', path: '/workflows' });
    const name = (state?.workflowName as string) || pathname.split('/').pop() || 'Workflow';
    crumbs.push({ label: name });
    return crumbs;
  }

  if (pathname === '/workflows') {
    return [{ label: 'Workflows' }];
  }

  if (pathname === '/events') {
    return [{ label: 'Events' }];
  }

  if (pathname === '/queues') {
    return [{ label: 'Queues' }];
  }

  if (pathname.startsWith('/sites/')) {
    crumbs.push({ label: 'Sites', path: '/sites' });
    const name = pathname.split('/').pop() || 'Site';
    crumbs.push({ label: decodeURIComponent(name) });
    return crumbs;
  }

  if (pathname === '/sites') {
    return [{ label: 'Sites' }];
  }

  if (pathname === '/submit') {
    return [{ label: 'Submit' }];
  }

  return [];
}

export function BreadcrumbNav() {
  const location = useLocation();
  const navigate = useNavigate();
  const crumbs = deriveCrumbs(location.pathname, location.state as Record<string, unknown> | null);

  // Don't render for top-level pages (single crumb)
  if (crumbs.length <= 1) {
    return null;
  }

  return (
    <Box sx={{ px: 3, py: 0.75, bgcolor: 'grey.100' }}>
      <Breadcrumbs
        separator={<NavigateNextIcon fontSize="small" />}
        aria-label="breadcrumb"
      >
        {crumbs.map((crumb, index) => {
          const isLast = index === crumbs.length - 1;

          if (isLast) {
            return (
              <Typography
                key={crumb.label}
                variant="body2"
                color="text.primary"
                sx={{ fontWeight: 500 }}
              >
                {crumb.label}
              </Typography>
            );
          }

          return (
            <Link
              key={crumb.label}
              component="button"
              variant="body2"
              color="text.secondary"
              underline="hover"
              onClick={() => {
                if (crumb.path) {
                  navigate(crumb.path);
                }
              }}
            >
              {crumb.label}
            </Link>
          );
        })}
      </Breadcrumbs>
    </Box>
  );
}
