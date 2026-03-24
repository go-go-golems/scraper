import { useState } from 'react';
import { Button } from '@mui/material';
import { ConfirmDialog } from '../common/ConfirmDialog';

interface RetryOpButtonProps {
  workflowId: string;
  opId: string;
  disabled?: boolean;
  onRetry: () => void;
  loading?: boolean;
}

export function RetryOpButton({
  workflowId: _workflowId,
  opId: _opId,
  disabled = false,
  onRetry,
  loading = false,
}: RetryOpButtonProps) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button
        variant="outlined"
        size="small"
        disabled={disabled}
        onClick={() => setOpen(true)}
      >
        Retry
      </Button>
      <ConfirmDialog
        open={open}
        title="Retry Operation"
        message="This will retry the failed operation from the beginning."
        confirmLabel="Retry"
        confirmColor="primary"
        loading={loading}
        onConfirm={() => {
          onRetry();
          setOpen(false);
        }}
        onCancel={() => setOpen(false)}
      />
    </>
  );
}
