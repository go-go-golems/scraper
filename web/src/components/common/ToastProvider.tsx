import {
  createContext,
  useCallback,
  useContext,
  useState,
  type ReactNode,
} from 'react';
import { Alert, type AlertColor, Snackbar, type SnackbarCloseReason } from '@mui/material';

interface ToastEntry {
  id: number;
  message: string;
  severity: AlertColor;
}

interface ToastContextValue {
  showToast: (message: string, severity?: AlertColor) => void;
}

const ToastContext = createContext<ToastContextValue>({
  showToast: () => {},
});

export function useToast() {
  return useContext(ToastContext);
}

const MAX_VISIBLE = 3;
const AUTO_DISMISS_MS = 4000;

interface ToastProviderProps {
  children: ReactNode;
}

let nextId = 0;

export function ToastProvider({ children }: ToastProviderProps) {
  const [toasts, setToasts] = useState<ToastEntry[]>([]);

  const showToast = useCallback((message: string, severity: AlertColor = 'info') => {
    const id = nextId++;
    setToasts((prev) => {
      const updated = [...prev, { id, message, severity }];
      // Keep only the last MAX_VISIBLE toasts
      return updated.slice(-MAX_VISIBLE);
    });

    // Auto-dismiss after timeout
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id));
    }, AUTO_DISMISS_MS);
  }, []);

  const handleClose = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {toasts.map((toast, index) => (
        <Snackbar
          key={toast.id}
          open
          anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
          sx={{
            bottom: `${24 + index * 56}px !important`,
            transition: 'bottom 200ms ease-in-out',
          }}
          onClose={(_event: unknown, reason: SnackbarCloseReason) => {
            if (reason === 'clickaway') return;
            handleClose(toast.id);
          }}
        >
          <Alert
            onClose={() => handleClose(toast.id)}
            severity={toast.severity}
            variant="filled"
            sx={{ width: '100%', maxWidth: 400 }}
          >
            {toast.message}
          </Alert>
        </Snackbar>
      ))}
    </ToastContext.Provider>
  );
}
