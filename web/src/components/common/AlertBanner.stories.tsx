import type { Meta, StoryObj } from '@storybook/react-vite';
import { AlertBanner } from './AlertBanner';

function DismissibleDemo() {
  return (
    <AlertBanner
      severity="error"
      message="7 ops failed in the last hour"
      action={{ label: 'View Failed Ops', onClick: () => alert('navigating…') }}
    />
  );
}

const meta: Meta<typeof AlertBanner> = {
  title: 'Common/AlertBanner',
  component: AlertBanner,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof AlertBanner>;

export const ErrorAlert: Story = {
  render: () => (
    <AlertBanner
      severity="error"
      message="7 ops failed in the last hour"
      action={{ label: 'View Failed Ops', onClick: () => {} }}
    />
  ),
};

export const WarningAlert: Story = {
  render: () => (
    <AlertBanner
      severity="warning"
      message="Queue site:hn:http at 95% capacity"
      action={{ label: 'View Queue', onClick: () => {} }}
    />
  ),
};

export const InfoAlert: Story = {
  render: () => (
    <AlertBanner
      severity="info"
      message="Engine restarted successfully"
      autoDismissMs={5000}
    />
  ),
};

export const NonDismissible: Story = {
  render: () => (
    <AlertBanner
      severity="error"
      message="Critical: scheduler is offline"
      dismissible={false}
    />
  ),
};

export const NoAction: Story = {
  render: () => (
    <AlertBanner
      severity="warning"
      message="3 workflows are stuck in pending state"
    />
  ),
};

export const Dismissed: Story = {
  render: () => <DismissibleDemo />,
};
