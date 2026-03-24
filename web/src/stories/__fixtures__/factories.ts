import type {
  EngineStatus,
  OpStatus,
  WorkflowListItem,
  WorkflowStatus,
  QueueStatus,
  WorkflowSummary,
  WorkflowOp,
  SiteSummary,
  VerbSummary,
} from '../../api/types';

export function createEngineStatus(overrides: Partial<EngineStatus> = {}): EngineStatus {
  return {
    Path: 'state/engine.db',
    Exists: true,
    Initialized: true,
    CurrentVersion: 2,
    LatestKnownMigration: 2,
    MigrationsUpToDate: true,
    WorkflowCount: 12,
    OpCounts: {
      pending: 3,
      ready: 23,
      running: 4,
      succeeded: 808,
      failed: 12,
      canceled: 0,
    },
    ActiveLeases: 4,
    ExpiredLeases: 0,
    ResultCount: 820,
    ArtifactCount: 1234,
    ...overrides,
  };
}

export function createEmptyEngineStatus(): EngineStatus {
  return createEngineStatus({
    WorkflowCount: 0,
    OpCounts: { pending: 0, ready: 0, running: 0, succeeded: 0, failed: 0, canceled: 0 },
    ActiveLeases: 0,
    ExpiredLeases: 0,
    ResultCount: 0,
    ArtifactCount: 0,
  });
}

export function createWorkflowListItem(overrides: {
  id?: string;
  site?: string;
  status?: WorkflowStatus;
  opTotal?: number;
  opDone?: number;
} = {}): WorkflowListItem {
  const id = overrides.id ?? 'wf-001';
  const status = overrides.status ?? 'running';
  return {
    workflow: {
      ID: id,
      Site: overrides.site ?? 'hackernews',
      Name: 'seed workflow',
      Status: status,
      Input: null,
      Metadata: {},
      CreatedAt: '2026-03-23T14:31:58Z',
      UpdatedAt: '2026-03-23T14:32:05Z',
    },
    opTotal: overrides.opTotal ?? 47,
    opDone: overrides.opDone ?? 12,
  };
}

export function createQueueStatus(overrides: Partial<QueueStatus> = {}): QueueStatus {
  return {
    site: 'hackernews',
    queue: 'site:hackernews:http',
    pending: 0,
    ready: 5,
    running: 2,
    succeeded: 120,
    failed: 1,
    inFlight: 2,
    maxInFlight: 4,
    ...overrides,
  };
}

export function createWorkflowSummary(overrides: {
  id?: string;
  site?: string;
  status?: WorkflowStatus;
} = {}): WorkflowSummary {
  return {
    workflow: {
      ID: overrides.id ?? 'wf-001',
      Site: overrides.site ?? 'hackernews',
      Name: 'seed workflow',
      Status: overrides.status ?? 'running',
      Input: null,
      Metadata: {},
      CreatedAt: '2026-03-23T14:31:58Z',
      UpdatedAt: '2026-03-23T14:32:05Z',
    },
    stats: {
      WorkflowID: overrides.id ?? 'wf-001',
      Total: 47,
      Pending: 3,
      Ready: 8,
      Running: 2,
      Succeeded: 32,
      Failed: 1,
      Canceled: 1,
    },
  };
}

export function createWorkflowOp(overrides: {
  id?: string;
  kind?: string;
  status?: OpStatus;
  queue?: string;
} = {}): WorkflowOp {
  return {
    op: {
      ID: overrides.id ?? 'wf-001:seed',
      WorkflowID: 'wf-001',
      Site: 'hackernews',
      Kind: overrides.kind ?? 'js',
      Queue: overrides.queue ?? 'site:hackernews:js',
      DedupKey: '',
      Input: { baseURL: 'https://news.ycombinator.com/', maxPages: 2 },
      DependsOn: [],
      Retry: { MaxAttempts: 3, BackoffKind: 'exponential', InitialBackoff: 1000000000, MaxBackoff: 30000000000, Multiplier: 2 },
      RetryState: { Attempt: 0, LastError: '' },
      Metadata: { script: 'seed.js' },
    },
    status: overrides.status ?? 'succeeded',
    createdAt: '2026-03-23T14:31:58Z',
    updatedAt: '2026-03-23T14:32:01Z',
  };
}

export function createSiteSummary(name: string = 'hackernews'): SiteSummary {
  return {
    name,
    databaseFileName: `${name}.db`,
    hasScripts: true,
    hasSubmitVerbs: true,
  };
}

export function createVerbSummary(overrides: { name?: string; site?: string } = {}): VerbSummary {
  const name = overrides.name ?? 'seed';
  const site = overrides.site ?? 'hackernews';
  return {
    name,
    fullPath: name,
    commandPath: `site ${site} run ${name}`,
    functionName: name,
    short: `Submit the ${site} ${name} workflow`,
    long: '',
    sections: [
      {
        slug: 'default',
        title: '',
        description: '',
        fields: [
          { name: 'base-url', type: 'string', help: 'Base URL', default: 'https://news.ycombinator.com/', required: false },
          { name: 'max-pages', type: 'int', help: 'Maximum pages to scrape', default: 1, required: false },
        ],
      },
    ],
  };
}
