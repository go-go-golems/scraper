import type { Meta, StoryObj } from '@storybook/react-vite';
import { ToastProvider, useToast } from './ToastProvider';
import { Button, Stack } from '@mui/material';

function ToastDemo() {
  const { showToast } = useToast();

  return (
    <Stack direction="row" spacing={2}>
      <Button variant="contained" color="success" onClick={() => showToast('Workflow submitted', 'success')}>
        Success Toast
      </Button>
      <Button variant="contained" color="error" onClick={() => showToast('Failed to cancel workflow', 'error')}>
        Error Toast
      </Button>
      <Button variant="contained" color="info" onClick={() => showToast('Refreshing data...', 'info')}>
        Info Toast
      </Button>
      <Button variant="contained" color="warning" onClick={() => showToast('Queue at 95% capacity', 'warning')}>
        Warning Toast
      </Button>
    </Stack>
  );
}

function StackedDemo() {
  const { showToast } = useToast();

  const fireMultiple = () => {
    showToast('First notification', 'info');
    setTimeout(() => showToast('Second notification', 'warning'), 300);
    setTimeout(() => showToast('Third notification', 'error'), 600);
  };

  return <Button variant="outlined" onClick={fireMultiple}>Fire 3 Toasts</Button>;
}

const meta: Meta<typeof ToastProvider> = {
  title: 'Common/ToastProvider',
  component: ToastProvider,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof ToastProvider>;

export const Success: Story = {
  render: () => (
    <ToastProvider>
      <ToastDemo />
    </ToastProvider>
  ),
};

export const Stacked: Story = {
  render: () => (
    <ToastProvider>
      <StackedDemo />
    </ToastProvider>
  ),
};
