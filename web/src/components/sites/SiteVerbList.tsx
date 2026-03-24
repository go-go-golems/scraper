import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Chip,
  Skeleton,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Typography,
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import type { VerbSummary } from '../../api/types';

interface SiteVerbListProps {
  verbs: VerbSummary[];
  loading: boolean;
}

export function SiteVerbList({ verbs, loading }: SiteVerbListProps) {
  if (loading) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} variant="rectangular" height={48} sx={{ borderRadius: 1 }} />
        ))}
      </Box>
    );
  }

  if (verbs.length === 0) {
    return (
      <Typography variant="body2" color="text.disabled" sx={{ py: 4, textAlign: 'center' }}>
        No verbs defined
      </Typography>
    );
  }

  return (
    <Box>
      {verbs.map((verb) => (
        <Accordion key={verb.name} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
              <Typography variant="subtitle2" sx={{ fontFamily: 'monospace' }}>
                {verb.name}
              </Typography>
              {verb.sections.length > 0 && (
                <Chip
                  label={`${verb.sections.reduce((sum, s) => sum + s.fields.length, 0)} fields`}
                  size="small"
                  variant="outlined"
                />
              )}
            </Box>
          </AccordionSummary>
          <AccordionDetails>
            {verb.short && (
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                {verb.short}
              </Typography>
            )}

            {verb.sections.map((section) => (
              <Box key={section.slug} sx={{ mb: 2 }}>
                {section.title && (
                  <Typography variant="caption" color="text.secondary" sx={{ mb: 1, display: 'block' }}>
                    {section.title}
                  </Typography>
                )}
                {section.fields.length > 0 && (
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell>Name</TableCell>
                        <TableCell>Type</TableCell>
                        <TableCell>Default</TableCell>
                        <TableCell>Help</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {section.fields.map((field) => (
                        <TableRow key={field.name}>
                          <TableCell>
                            <Typography
                              variant="body2"
                              sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
                            >
                              {field.name}
                              {field.required && (
                                <Typography component="span" color="error.main" sx={{ ml: 0.5 }}>
                                  *
                                </Typography>
                              )}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Chip label={field.type} size="small" variant="outlined" />
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2" color="text.secondary" sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}>
                              {field.default !== undefined ? String(field.default) : '-'}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2" color="text.secondary">
                              {field.help ?? '-'}
                            </Typography>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </Box>
            ))}
          </AccordionDetails>
        </Accordion>
      ))}
    </Box>
  );
}
