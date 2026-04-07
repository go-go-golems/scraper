// Domain types matching Go engine/model/types.go

export type WorkflowStatus = 'pending' | 'running' | 'succeeded' | 'failed' | 'canceled';
export type OpStatus = 'pending' | 'ready' | 'running' | 'succeeded' | 'failed' | 'canceled';

export interface WorkflowRun {
  ID: string;
  Site: string;
  Name: string;
  Status: WorkflowStatus;
  Input: unknown;
  Metadata: Record<string, string>;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface OpSpec {
  ID: string;
  WorkflowID: string;
  ParentID?: string;
  Site: string;
  Kind: string;
  Queue: string;
  DedupKey: string;
  Input: unknown;
  DependsOn: Dependency[];
  Retry: RetryPolicy;
  RetryState: RetryState;
  Metadata: Record<string, string>;
}

export interface Dependency {
  OpID: string;
  Required: boolean;
}

export interface RetryPolicy {
  MaxAttempts: number;
  BackoffKind: string;
  InitialBackoff: number;
  MaxBackoff: number;
  Multiplier: number;
}

export interface RetryState {
  Attempt: number;
  NextAttemptAt?: string;
  LastError: string;
}

export interface Lease {
  WorkerID: string;
  Token: string;
  AcquiredAt: string;
  ExpiresAt: string;
}

export interface OpError {
  Code: string;
  Message: string;
  Retryable: boolean;
  Details: unknown;
  OccurredAt: string;
}

export interface OpResult {
  OpID: string;
  Data: unknown;
  Records: { Collection: string; Key: string; Data: unknown }[];
  Artifacts: { ID: string; Name: string; Kind: string; ContentType: string }[];
  EmittedIDs: string[];
  Error?: OpError;
  CompletedAt: string;
}

// API response types matching Go api/types

export interface EngineStatus {
  Path: string;
  Exists: boolean;
  Initialized: boolean;
  CurrentVersion: number;
  LatestKnownMigration: number;
  MigrationsUpToDate: boolean;
  WorkflowCount: number;
  OpCounts: Record<OpStatus, number>;
  ActiveLeases: number;
  ExpiredLeases: number;
  ResultCount: number;
  ArtifactCount: number;
}

export interface WorkflowListItem {
  workflow: WorkflowRun;
  opTotal: number;
  opDone: number;
}

export interface WorkflowStats {
  WorkflowID: string;
  Total: number;
  Pending: number;
  Ready: number;
  Running: number;
  Succeeded: number;
  Failed: number;
  Canceled: number;
}

export interface WorkflowSummary {
  workflow: WorkflowRun;
  stats: WorkflowStats;
}

export interface WorkflowOp {
  op: OpSpec;
  status: OpStatus;
  nextAttemptAt?: string;
  createdAt: string;
  updatedAt: string;
  lease?: Lease;
}

export interface QueueStatus {
  site: string;
  queue: string;
  pending: number;
  ready: number;
  running: number;
  succeeded: number;
  failed: number;
  inFlight: number;
  maxInFlight: number;
  tokens?: number;
  burst?: number;
  ratePerSecond?: number;
}

export interface SiteSummary {
  name: string;
  databaseFileName: string;
  hasScripts: boolean;
  hasSubmitVerbs: boolean;
}

export interface VerbSummary {
  name: string;
  fullPath: string;
  commandPath: string;
  functionName: string;
  short: string;
  long: string;
  sections: SectionSummary[];
}

export interface SectionSummary {
  slug: string;
  title: string;
  description: string;
  fields: FieldSummary[];
}

export interface FieldSummary {
  name: string;
  type: string;
  help?: string;
  default?: unknown;
  choices?: string[];
  required: boolean;
}

export interface SiteDetail {
  name: string;
  databaseFileName: string;
  hasScripts: boolean;
  hasSubmitVerbs: boolean;
  verbCount: number;
  scriptCount: number;
  scripts: string[];
  queuePolicies: QueuePolicySummary[];
}

export interface QueuePolicySummary {
  queue: string;
  maxInFlight: number;
  rateLimit?: {
    kind: string;
    ratePerSecond: number;
    burst: number;
  };
}

export interface ArtifactSummary {
  id: string;
  opID: string;
  workflowID: string;
  name: string;
  kind: string;
  contentType: string;
  metadata?: Record<string, string>;
  size: number;
  createdAt: string;
  previewable: boolean;
  previewKind?: string;
}

export interface WorkflowArtifactListResponse {
  workflowID: string;
  total: number;
  artifacts: ArtifactSummary[];
}
