import type { Meta, StoryObj } from '@storybook/react';
import { OpRuntimeTab } from './OpRuntimeTab';
import { mockEvent, resetMockIds } from '../../../test-utils/mockRuntimeEvents';
import {
  RuntimeEventSeverity,
  RuntimeEventSource,
  RuntimeEventKind,
} from '../../../pb/proto/scraper/runtime/v1/events_pb';

const meta: Meta<typeof OpRuntimeTab> = {
  title: 'Workflows/OpDetail/OpRuntimeTab',
  component: OpRuntimeTab,
};

export default meta;
type Story = StoryObj<typeof OpRuntimeTab>;

export const Empty: Story = {
  args: {
    events: [],
    loading: false,
    connectionState: 'live',
  },
};

export const Loading: Story = {
  args: {
    events: [],
    loading: true,
    connectionState: 'connecting',
  },
};

export const WithEvents: Story = {
  args: {
    events: (() => {
      resetMockIds();
      return [
        mockEvent({ id: 'ev-1', message: 'Op leased by worker-01', severity: RuntimeEventSeverity.INFO, source: RuntimeEventSource.WORKER, kind: RuntimeEventKind.OP_LEASED }),
        mockEvent({ id: 'ev-2', message: 'Fetching https://news.ycombinator.com/', severity: RuntimeEventSeverity.INFO, source: RuntimeEventSource.RUNNER, kind: RuntimeEventKind.OP_SUCCEEDED }),
        mockEvent({ id: 'ev-3', message: 'Op succeeded (420ms)', severity: RuntimeEventSeverity.INFO, source: RuntimeEventSource.SCHEDULER, kind: RuntimeEventKind.OP_SUCCEEDED }),
      ];
    })(),
    loading: false,
    connectionState: 'live',
  },
};

export const WithError: Story = {
  args: {
    events: (() => {
      resetMockIds();
      return [
        mockEvent({ id: 'ev-1', message: 'Op leased by worker-01', severity: RuntimeEventSeverity.INFO, source: RuntimeEventSource.WORKER, kind: RuntimeEventKind.OP_LEASED }),
        mockEvent({ id: 'ev-2', message: 'Connection refused', severity: RuntimeEventSeverity.ERROR, source: RuntimeEventSource.RUNNER, kind: RuntimeEventKind.OP_FAILED }),
      ];
    })(),
    loading: false,
    connectionState: 'error',
  },
};

export const Disconnected: Story = {
  args: {
    events: (() => {
      resetMockIds();
      return [
        mockEvent({ id: 'ev-1', message: 'Cached event', severity: RuntimeEventSeverity.INFO, source: RuntimeEventSource.WORKER, kind: RuntimeEventKind.OP_LEASED }),
      ];
    })(),
    loading: false,
    connectionState: 'closed',
  },
};
