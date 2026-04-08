import { Box, Card, CardActionArea, CardContent, Chip, Typography } from '@mui/material';
import type { SiteDetail } from '../../api/types';
import { SiteOriginChip } from './SiteOriginChip';

interface SiteCardProps {
  site: SiteDetail;
  onClick: () => void;
}

export function SiteCard({ site, onClick }: SiteCardProps) {
  return (
    <Card
      sx={{
        '&:hover': { boxShadow: 4 },
        transition: 'box-shadow 0.2s',
      }}
    >
      <CardActionArea onClick={onClick}>
        <CardContent>
          <Typography variant="h6" gutterBottom>
            {site.name}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1.5 }}>
            {site.databaseFileName}
          </Typography>

          <Box sx={{ display: 'flex', gap: 1, mb: 1.5, flexWrap: 'wrap' }}>
            <SiteOriginChip originKind={site.originKind} />
            <Chip label={`${site.verbCount} verbs`} size="small" color="primary" variant="outlined" />
            <Chip label={`${site.scriptCount} scripts`} size="small" color="secondary" variant="outlined" />
          </Box>

          {site.manifestPath ? (
            <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mb: 1.5 }}>
              Manifest: {site.manifestPath}
            </Typography>
          ) : null}

          {(site.queuePolicies ?? []).length > 0 ? (
            <Box>
              {(site.queuePolicies ?? []).map((p) => (
                <Typography key={p.queue} variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                  {p.queue}: MaxInFlight: {p.maxInFlight}
                  {p.rateLimit ? `, Rate: ${p.rateLimit.ratePerSecond}/sec` : ''}
                </Typography>
              ))}
            </Box>
          ) : (
            <Typography variant="caption" color="text.secondary">
              Default policy (max 1 in-flight)
            </Typography>
          )}
        </CardContent>
      </CardActionArea>
    </Card>
  );
}
