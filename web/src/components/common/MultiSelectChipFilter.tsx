import Autocomplete from '@mui/material/Autocomplete';
import Chip from '@mui/material/Chip';
import TextField from '@mui/material/TextField';
import type { ChipProps } from '@mui/material';

export interface MultiSelectOption {
  value: string;
  label: string;
  color?: ChipProps['color'];
}

interface MultiSelectChipFilterProps {
  label: string;
  options: MultiSelectOption[];
  selected: string[];
  onChange: (selected: string[]) => void;
  disabled?: boolean;
  size?: 'small' | 'medium';
}

export function MultiSelectChipFilter({
  label,
  options,
  selected,
  onChange,
  disabled = false,
  size = 'small',
}: MultiSelectChipFilterProps) {
  const selectedOptions = options.filter((o) => selected.includes(o.value));

  return (
    <Autocomplete
      multiple
      size={size}
      disabled={disabled}
      options={options}
      getOptionLabel={(option) => option.label}
      value={selectedOptions}
      onChange={(_event, newValue) => {
        onChange(newValue.map((v) => v.value));
      }}
      renderTags={(value, getTagProps) =>
        value.map((option, index) => {
          const { key, ...rest } = getTagProps({ index });
          return (
            <Chip
              key={key}
              label={option.label}
              color={option.color ?? 'default'}
              size={size}
              {...rest}
            />
          );
        })
      }
      renderInput={(params) => (
        <TextField
          {...params}
          label={selected.length === 0 ? `${label} (all)` : label}
          placeholder={selected.length === 0 ? '' : 'Add…'}
        />
      )}
      sx={{ minWidth: 220 }}
    />
  );
}
