import { useState } from 'react';
import { Button } from '@mui/material';
import type { WorkflowStatus } from '../../api/types';
import { ConfirmDialog } from '../common/ConfirmDialog';

interface CancelWorkflowButtonProps {
  workflowId: string;
  status: WorkflowStatus;
  onCancel: () => void;
  loading?: boolean;
}

export function CancelWorkflowButton({
  workflowId: _workflowId,
  status,
  onCancel,
  loading = false,
}: CancelWorkflowButtonProps) {
  const [open, setOpen] = useState(false);

  if (status !== 'pending' && status !== 'running') {
    return null;
  }

  return (
    <>
      <Button
        variant="outlined"
        color="error"
        size="small"
        onClick={() => setOpen(true)}
      >
        Cancel Workflow
      </Button>
      <ConfirmDialog
        open={open}
        title="Cancel Workflow"
        message="Are you sure you want to cancel this workflow? This action cannot be undone."
        confirmLabel="Cancel Workflow"
        confirmColor="error"
        loading={loading}
        onConfirm={() => {
          onCancel();
          setOpen(false);
        }}
        onCancel={() => setOpen(false)}
      />
    </>
  );
}
