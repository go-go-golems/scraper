import type { Meta, StoryObj } from '@storybook/react-vite';
import { RuntimeEventTable } from './RuntimeEventTable';
import type { RuntimeEventV1 } from '../../pb/proto/scraper/runtime/v1/events_pb';
import {
  RuntimeEventKind,
  RuntimeEventSeverity,
  RuntimeEventSource,
} from '../../pb/proto/scraper/runtime/v1/events_pb';

// ── mock data factory ────────────────────────────────────────────────

let mockId = 0;

function mockEvent(overrides: Partial<RuntimeEventV1> = {}): RuntimeEventV1 {
  const id = `evt-${++mockId}`;
  const base: RuntimeEventV1 = {
    id,
    occurredAt: {
      seconds: BigInt(Math.floor(Date.now() / 1000) - mockId * 30),
      nanos: 0,
    },
    source: RuntimeEventSource.WORKER,
    severity: RuntimeEventSeverity.INFO,
    kind: RuntimeEventKind.OP_COMPLETED,
    message: `Event ${id}`,
    workflowId: 'wf-test-001',
    opId: `op-${mockId}`,
    site: 'hackernews',
    workerId: 'worker-01',
    payload: {},
  } as RuntimeEventV1;
  return { ...base, ...overrides };
}

function generateEvents(count: number): RuntimeEventV1[] {
  mockId = 0;
  const severities = [RuntimeEventSeverity.DEBUG, RuntimeEventSeverity.INFO, RuntimeEventSeverity.WARN, RuntimeEventSeverity.ERROR];
  const sources = [RuntimeEventSource.SCHEDULER, RuntimeEventSource.WORKER, RuntimeEventSource.RUNNER, RuntimeEventSource.SERVER];
  const kinds = [RuntimeEventKind.WORKFLOW_STARTED, RuntimeEventKind.OP_DISPATCHED, RuntimeEventKind.OP_COMPLETED, RuntimeEventKind.OP_FAILED, RuntimeEventKind.OP_ERROR];
  const messages = [
    'Workflow started successfully',
    'Op dispatched to worker-01',
    'Operation completed in 342ms',
    'Connection timeout exceeded',
    'HTTP 503 from upstream server',
    'Retry attempt 2/3',
    'Artifact stored: page-1.html',
    'Token bucket refreshed',
    'Queue drained, no pending ops',
    'Script execution completed',
  ];
  return Array.from({ length: count }, (_, i) =>
    mockEvent({
      source: sources[i % sources.length],
      severity: severities[i % severities.length],
      kind: kinds[i % kinds.length],
      message: messages[i % messages.length],
      payload: i % 3 === 0 ? { durationMillis: 100 + i * 50, attempt: 1 } : {},
    }),
  );
}

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
    events: generateEvents(20),
  },
};

export const ExpandedRow: Story = {
  args: {
    events: generateEvents(20),
  },
  play: async ({ canvas }) => {
    // Click the 3rd row to expand it
    const rows = canvas.getAllByRole('row');
    if (rows.length > 3) {
      rows[3].click();
    }
  },
};

export const WithFilters: Story = {
  args: {
    events: generateEvents(20),
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
    events: generateEvents(60),
    showPagination: true,
  },
};

export const NonDense: Story = {
  args: {
    events: generateEvents(10),
    dense: false,
  },
};
