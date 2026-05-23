import type { Meta, StoryObj } from '@storybook/react-vite';
import { RuntimeEventTable } from './RuntimeEventTable';
import { generateMockEvents } from '../../test-utils/mockRuntimeEvents';

const meta: Meta<typeof RuntimeEventTable> = {
  title: 'Workflows/RuntimeEventTable',
  component: RuntimeEventTable,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof RuntimeEventTable>;

export const Empty: Story = {
  args: {
    events: [],
    emptyMessage: 'No runtime events matched the current filters.',
  },
};

export const WithEvents: Story = {
  args: {
    events: generateMockEvents(20),
  },
};

export const ExpandedRow: Story = {
  args: {
    events: generateMockEvents(20),
  },
  play: async ({ canvasElement }) => {
    // Click the 3rd row to expand it.
    const rows = canvasElement.querySelectorAll('[role="row"]');
    if (rows.length > 3) {
      (rows[3] as HTMLElement).click();
    }
  },
};

export const WithFilters: Story = {
  args: {
    events: generateMockEvents(20),
  },
};

export const Loading: Story = {
  args: {
    events: [],
    loading: true,
  },
};

export const WithPagination: Story = {
  args: {
    events: generateMockEvents(60),
    showPagination: true,
  },
};

export const NonDense: Story = {
  args: {
    events: generateMockEvents(10),
    dense: false,
  },
};
