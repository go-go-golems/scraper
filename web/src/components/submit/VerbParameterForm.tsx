import {
  Box,
  FormControl,
  FormControlLabel,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import type { SelectChangeEvent } from '@mui/material';
import type { VerbSummary, FieldSummary } from '../../api/types';

interface VerbParameterFormProps {
  verb: VerbSummary;
  values: Record<string, unknown>;
  onChange: (field: string, value: unknown) => void;
}

function renderField(
  field: FieldSummary,
  value: unknown,
  onChange: (field: string, value: unknown) => void,
) {
  const currentValue = value ?? field.default ?? '';

  // Field with choices -> Select
  if (field.choices && field.choices.length > 0) {
    return (
      <FormControl key={field.name} fullWidth size="small" required={field.required}>
        <InputLabel id={`field-${field.name}-label`}>{field.name}</InputLabel>
        <Select
          labelId={`field-${field.name}-label`}
          value={String(currentValue)}
          label={field.name}
          onChange={(e: SelectChangeEvent) => onChange(field.name, e.target.value)}
        >
          {field.choices.map((choice) => (
            <MenuItem key={choice} value={choice}>
              {choice}
            </MenuItem>
          ))}
        </Select>
        {field.help && (
          <Typography variant="caption" color="text.secondary" sx={{ mt: 0.5, ml: 1.75 }}>
            {field.help}
          </Typography>
        )}
      </FormControl>
    );
  }

  // Bool -> Switch
  if (field.type === 'bool') {
    return (
      <FormControlLabel
        key={field.name}
        control={
          <Switch
            checked={Boolean(currentValue)}
            onChange={(_e, checked) => onChange(field.name, checked)}
            size="small"
          />
        }
        label={
          <Box>
            <Typography variant="body2">{field.name}</Typography>
            {field.help && (
              <Typography variant="caption" color="text.secondary">
                {field.help}
              </Typography>
            )}
          </Box>
        }
      />
    );
  }

  // int / float -> TextField type="number"
  if (field.type === 'int' || field.type === 'float') {
    return (
      <TextField
        key={field.name}
        label={field.name}
        type="number"
        size="small"
        fullWidth
        required={field.required}
        helperText={field.help}
        value={currentValue === '' ? '' : currentValue}
        inputProps={field.type === 'int' ? { step: 1 } : undefined}
        onChange={(e) => {
          const raw = e.target.value;
          if (raw === '') {
            onChange(field.name, '');
            return;
          }
          const parsed = field.type === 'int' ? parseInt(raw, 10) : parseFloat(raw);
          onChange(field.name, isNaN(parsed) ? raw : parsed);
        }}
      />
    );
  }

  // Default: string -> TextField
  return (
    <TextField
      key={field.name}
      label={field.name}
      size="small"
      fullWidth
      required={field.required}
      helperText={field.help}
      value={String(currentValue)}
      onChange={(e) => onChange(field.name, e.target.value)}
    />
  );
}

export function VerbParameterForm({ verb, values, onChange }: VerbParameterFormProps) {
  const allFields = verb.sections.flatMap((s) => s.fields);

  if (allFields.length === 0) {
    return (
      <Typography variant="body2" color="text.secondary">
        This verb has no parameters.
      </Typography>
    );
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      {verb.sections.map((section) => (
        <Box key={section.slug} sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
          {section.title && (
            <Typography variant="subtitle2" color="text.secondary">
              {section.title}
            </Typography>
          )}
          {section.description && (
            <Typography variant="caption" color="text.secondary">
              {section.description}
            </Typography>
          )}
          {section.fields.map((field) => renderField(field, values[field.name], onChange))}
        </Box>
      ))}
    </Box>
  );
}
