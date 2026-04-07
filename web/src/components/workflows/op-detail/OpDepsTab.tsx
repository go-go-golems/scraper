import {
  Box,
  Chip,
  List,
  ListItem,
  ListItemText,
  Typography,
} from '@mui/material';
import type { Dependency } from '../../../api/types';

interface OpDepsTabProps {
  dependsOn: Dependency[];
}

export function OpDepsTab({ dependsOn }: OpDepsTabProps) {
  if (dependsOn.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        No dependencies
      </Typography>
    );
  }

  return (
    <List dense disablePadding>
      {dependsOn.map((dep) => (
        <ListItem key={dep.OpID} disablePadding sx={{ py: 0.25 }}>
          <ListItemText
            primary={
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                <Typography
                  variant="body2"
                  sx={{ fontFamily: 'monospace', fontSize: '0.8rem' }}
                >
                  {dep.OpID}
                </Typography>
                {dep.Required && (
                  <Chip
                    label="required"
                    size="small"
                    color="primary"
                    variant="outlined"
                  />
                )}
              </Box>
            }
          />
        </ListItem>
      ))}
    </List>
  );
}
