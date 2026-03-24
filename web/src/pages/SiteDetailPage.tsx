import { useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Chip,
  IconButton,
  Tab,
  Tabs,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import { useGetSiteDetailQuery, useListVerbsQuery } from '../api/catalogApi';
import { SiteVerbList } from '../components/sites/SiteVerbList';
import { SiteScriptBrowser } from '../components/sites/SiteScriptBrowser';

export function SiteDetailPage() {
  const { siteName } = useParams<{ siteName: string }>();
  const navigate = useNavigate();
  const [tabIndex, setTabIndex] = useState(0);

  const { data: detail, isLoading: detailLoading } = useGetSiteDetailQuery(siteName!, {
    skip: !siteName,
  });

  const { data: verbs = [], isLoading: verbsLoading } = useListVerbsQuery(siteName!, {
    skip: !siteName,
  });

  if (!siteName) {
    return <Typography color="text.disabled">No site name</Typography>;
  }

  if (detailLoading) {
    return <Typography color="text.secondary">Loading site...</Typography>;
  }

  if (!detail) {
    return <Typography color="text.disabled">Site not found</Typography>;
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {/* Back navigation */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
        <IconButton onClick={() => navigate('/sites')} size="small">
          <ArrowBackIcon />
        </IconButton>
        <Typography variant="body2" color="text.secondary">
          Back to Sites
        </Typography>
      </Box>

      {/* Header */}
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, flexWrap: 'wrap' }}>
        <Typography variant="h5" component="h1">
          {detail.name}
        </Typography>
        <Typography variant="body2" color="text.secondary">
          {detail.databaseFileName}
        </Typography>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Chip label={`${detail.verbCount} verbs`} size="small" color="primary" variant="outlined" />
          <Chip label={`${detail.scriptCount} scripts`} size="small" color="secondary" variant="outlined" />
        </Box>
      </Box>

      {/* Tabs */}
      <Card>
        <Tabs value={tabIndex} onChange={(_, v: number) => setTabIndex(v)}>
          <Tab label="Overview" />
          <Tab label="Verbs" />
          <Tab label="Scripts" />
        </Tabs>

        <CardContent>
          {/* Overview tab */}
          {tabIndex === 0 && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Typography variant="subtitle2">Queue Policies</Typography>
              {detail.queuePolicies.length > 0 ? (
                <TableContainer>
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Queue</TableCell>
                        <TableCell align="right">Max In-Flight</TableCell>
                        <TableCell align="right">Rate/sec</TableCell>
                        <TableCell align="right">Burst</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {detail.queuePolicies.map((p) => (
                        <TableRow key={p.queue}>
                          <TableCell>
                            <Typography variant="body2" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                              {p.queue}
                            </Typography>
                          </TableCell>
                          <TableCell align="right">{p.maxInFlight}</TableCell>
                          <TableCell align="right">
                            {p.rateLimit ? p.rateLimit.ratePerSecond : (
                              <Typography component="span" variant="body2" color="text.disabled">
                                none
                              </Typography>
                            )}
                          </TableCell>
                          <TableCell align="right">
                            {p.rateLimit ? p.rateLimit.burst : (
                              <Typography component="span" variant="body2" color="text.disabled">
                                none
                              </Typography>
                            )}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              ) : (
                <Typography variant="body2" color="text.secondary">
                  Default policy (max 1 in-flight, no rate limit)
                </Typography>
              )}

              <Typography variant="subtitle2" sx={{ mt: 2 }}>
                Stats
              </Typography>
              <Box sx={{ display: 'flex', gap: 3 }}>
                <Typography variant="body2">
                  Verbs: <strong>{detail.verbCount}</strong>
                </Typography>
                <Typography variant="body2">
                  Scripts: <strong>{detail.scriptCount}</strong>
                </Typography>
                <Typography variant="body2">
                  Has submit verbs: <strong>{detail.hasSubmitVerbs ? 'Yes' : 'No'}</strong>
                </Typography>
              </Box>
            </Box>
          )}

          {/* Verbs tab */}
          {tabIndex === 1 && (
            <SiteVerbList verbs={verbs} loading={verbsLoading} />
          )}

          {/* Scripts tab */}
          {tabIndex === 2 && (
            detail.scripts.length > 0 ? (
              <SiteScriptBrowser site={siteName} scripts={detail.scripts} />
            ) : (
              <Typography variant="body2" color="text.disabled" sx={{ py: 4, textAlign: 'center' }}>
                No scripts found
              </Typography>
            )
          )}
        </CardContent>
      </Card>
    </Box>
  );
}
