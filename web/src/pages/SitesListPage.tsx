import { Box, Grid, Skeleton, Typography } from '@mui/material';
import { useNavigate } from 'react-router-dom';
import { SiteCard } from '../components/sites/SiteCard';
import { useListSitesQuery, useGetSiteDetailQuery } from '../api/catalogApi';
import type { SiteSummary } from '../api/types';

function SiteCardWithDetail({ site, onClick }: { site: SiteSummary; onClick: () => void }) {
  const { data: detail, isLoading } = useGetSiteDetailQuery(site.name);

  if (isLoading || !detail) {
    return (
      <Box sx={{ p: 2 }}>
        <Skeleton variant="text" width={120} height={32} />
        <Skeleton variant="text" width={180} />
        <Skeleton variant="rectangular" height={60} sx={{ mt: 1, borderRadius: 1 }} />
      </Box>
    );
  }

  return <SiteCard site={detail} onClick={onClick} />;
}

export function SitesListPage() {
  const { data: sites = [], isLoading } = useListSitesQuery();
  const navigate = useNavigate();

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 3 }}>
      <Typography variant="h5" component="h1">
        Sites
      </Typography>

      {isLoading && (
        <Grid container spacing={3}>
          {Array.from({ length: 3 }).map((_, i) => (
            <Grid key={i} size={{ xs: 12, sm: 6, md: 4 }}>
              <Skeleton variant="rectangular" height={180} sx={{ borderRadius: 1 }} />
            </Grid>
          ))}
        </Grid>
      )}

      {!isLoading && sites.length === 0 && (
        <Typography variant="body2" color="text.disabled" sx={{ py: 4, textAlign: 'center' }}>
          No sites found
        </Typography>
      )}

      {!isLoading && sites.length > 0 && (
        <Grid container spacing={3}>
          {sites.map((site) => (
            <Grid key={site.name} size={{ xs: 12, sm: 6, md: 4 }}>
              <SiteCardWithDetail
                site={site}
                onClick={() => navigate(`/sites/${site.name}`)}
              />
            </Grid>
          ))}
        </Grid>
      )}
    </Box>
  );
}
