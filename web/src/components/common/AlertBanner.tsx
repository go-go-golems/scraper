import { useState, useEffect, useCallback } from 'react';
import Alert from '@mui/material/Alert';
import Button from '@mui/material/Button';
import Collapse from '@mui/material/Collapse';

interface AlertBannerProps {
  severity: 'error' | 'warning' | 'info';
  message: string;
  action?: {
    label: string;
    onClick: () => void;
  };
  dismissible?: boolean;
  autoDismissMs?: number | null;
}

export function AlertBanner({
  severity,
  message,
  action,
  dismissible = true,
  autoDismissMs = null,
}: AlertBannerProps) {
  const [visible, setVisible] = useState(true);

  const handleDismiss = useCallback(() => {
    setVisible(false);
  }, []);

  useEffect(() => {
    if (!autoDismissMs) return;
    const timer = setTimeout(handleDismiss, autoDismissMs);
    return () => clearTimeout(timer);
  }, [autoDismissMs, handleDismiss]);

  if (!visible) return null;

  const alertVariant = severity === 'error' ? 'filled' : 'standard';

  return (
    <Collapse in={visible}>
      <Alert
        severity={severity}
        variant={alertVariant}
        onClose={dismissible ? handleDismiss : undefined}
        action={
          action ? (
            <Button
              color="inherit"
              size="small"
              onClick={action.onClick}
            >
              {action.label}
            </Button>
          ) : undefined
        }
        sx={{ mb: 2 }}
      >
        {message}
      </Alert>
    </Collapse>
  );
}
