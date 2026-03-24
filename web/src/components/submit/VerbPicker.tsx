import {
  FormControl,
  FormControlLabel,
  Radio,
  RadioGroup,
  Skeleton,
  Typography,
  Box,
} from '@mui/material';
import type { VerbSummary } from '../../api/types';

interface VerbPickerProps {
  verbs: VerbSummary[];
  selected: string | null;
  onSelect: (verb: string) => void;
  loading: boolean;
}

export function VerbPicker({ verbs, selected, onSelect, loading }: VerbPickerProps) {
  if (loading) {
    return (
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} width={200} height={32} />
        ))}
      </Box>
    );
  }

  return (
    <FormControl>
      <RadioGroup
        value={selected ?? ''}
        onChange={(_event, value) => {
          onSelect(value);
        }}
      >
        {verbs.map((verb) => (
          <FormControlLabel
            key={verb.name}
            value={verb.name}
            control={<Radio size="small" />}
            label={
              <Box>
                <Typography variant="body2" component="span" sx={{ fontWeight: 500 }}>
                  {verb.name}
                </Typography>
                {verb.short && (
                  <Typography
                    variant="caption"
                    color="text.secondary"
                    component="span"
                    sx={{ ml: 1 }}
                  >
                    {verb.short}
                  </Typography>
                )}
              </Box>
            }
          />
        ))}
      </RadioGroup>
    </FormControl>
  );
}
