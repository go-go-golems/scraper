import type { Meta, StoryObj } from '@storybook/react-vite';
import { AppErrorBoundary } from './AppErrorBoundary';

function ThrowingChild({ shouldThrow }: { shouldThrow: boolean }) {
  if (shouldThrow) {
    throw new Error('Simulated error: something exploded in a component');
  }
  return <div>Everything is fine — no errors here.</div>;
}

const meta: Meta<typeof AppErrorBoundary> = {
  title: 'Common/AppErrorBoundary',
  component: AppErrorBoundary,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof AppErrorBoundary>;

export const ErrorState: Story = {
  render: () => (
    <AppErrorBoundary>
      <ThrowingChild shouldThrow={true} />
    </AppErrorBoundary>
  ),
};

export const Healthy: Story = {
  render: () => (
    <AppErrorBoundary>
      <ThrowingChild shouldThrow={false} />
    </AppErrorBoundary>
  ),
};
