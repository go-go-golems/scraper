import type {
  ArtifactSummary,
  EngineStatus,
  OpStatus,
  OpResult,
  WorkflowListItem,
  WorkflowStatus,
  QueueStatus,
  WorkflowSummary,
  WorkflowOp,
  SiteDetail,
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
    originKind: 'manifest',
    manifestPath: 'site.yaml',
    hasScripts: true,
    hasSubmitVerbs: true,
  };
}

export function createSiteDetail(
  name: string = 'hackernews',
  overrides: Partial<Omit<SiteDetail, 'name'>> = {},
): SiteDetail {
  return {
    name,
    databaseFileName: `${name}.db`,
    originKind: 'manifest',
    manifestPath: 'site.yaml',
    hasScripts: true,
    hasSubmitVerbs: true,
    verbCount: 2,
    scriptCount: 3,
    scripts: ['seed.js', 'detail.js', 'export.js'],
    queuePolicies: [],
    ...overrides,
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

export function createOpResult(overrides: {
  opId?: string;
  data?: unknown;
  artifacts?: { id: string; name: string; kind: string; contentType: string }[];
  emittedIds?: string[];
  error?: { code: string; message: string; retryable: boolean };
} = {}): OpResult {
  return {
    OpID: overrides.opId ?? 'wf-001:seed',
    Data: overrides.data ?? null,
    Records: [],
    Artifacts: (overrides.artifacts ?? []).map((a) => ({
      ID: a.id,
      Name: a.name,
      Kind: a.kind,
      ContentType: a.contentType,
    })),
    EmittedIDs: overrides.emittedIds ?? [],
    ...(overrides.error
      ? {
          Error: {
            Code: overrides.error.code,
            Message: overrides.error.message,
            Retryable: overrides.error.retryable,
            Details: null,
            OccurredAt: '2026-03-23T14:32:10Z',
          },
        }
      : {}),
    CompletedAt: '2026-03-23T14:32:05Z',
  };
}

export function createArtifactSummary(overrides: Partial<ArtifactSummary> = {}): ArtifactSummary {
  return {
    id: 'art-001',
    opID: 'wf-001:seed',
    workflowID: 'wf-001',
    name: 'page.html',
    kind: 'page',
    contentType: 'text/html',
    size: 24_320,
    createdAt: '2026-03-23T14:32:05Z',
    ...overrides,
  };
}

const logMessages = [
  'Starting JS extraction for hackernews',
  'Loaded seed.js script (2.4 KB)',
  'Navigating to https://news.ycombinator.com/',
  'Page loaded, running extraction function',
  'Querying DOM: document.querySelectorAll(".athing")',
  'Found 30 items on page 1',
  'Extracting title, URL, score for each item',
  'Processing item 1/30: Show HN: A cool project',
  'Processing item 15/30: Ask HN: Best practices',
  'Processing item 30/30: New in Go 1.24',
  'All items extracted successfully',
  'Checking for pagination link',
  'Found next page: /news?p=2',
  'Emitting 30 child ops for detail pages',
  'Extraction complete',
  'Writing 30 records to collection "items"',
  'Saving page HTML as artifact (24.3 KB)',
  'Saving extracted data as artifact (5.1 KB)',
  'Op finished in 6.8s',
  'Cleaning up resources',
];

export function createLogEntries(count: number): { timestamp: string; message: string }[] {
  return Array.from({ length: count }, (_, i) => {
    const seconds = Math.floor(i * 0.45);
    const ms = String(Math.floor((i * 450) % 1000)).padStart(3, '0');
    const s = String(58 + seconds).padStart(2, '0');
    return {
      timestamp: `2026-03-23T14:31:${s}.${ms}Z`,
      message: logMessages[i % logMessages.length],
    };
  });
}
