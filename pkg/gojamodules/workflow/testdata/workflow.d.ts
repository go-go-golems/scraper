declare module "workflow" {
  export interface ValueRef<T = unknown> { readonly schema?: string }
  export interface SetRef<T = unknown> { readonly itemSchema?: string }
  export interface JobRef<T = unknown> {
    output(name: string): ValueRef<T>;
  }
  export type BudgetDimension =
    | "requests" | "input_tokens" | "output_tokens"
    | "embedding_tokens" | "input_bytes" | "output_bytes"
    | "cost_microunits";
  export type BudgetAmounts = Partial<Record<BudgetDimension, number>>;
  export interface BudgetClaim {
    account: string;
    reserve: BudgetAmounts;
    onExhausted: "fail-run" | "block" | "require-approval";
    approvalGate?: string;
  }
  export interface IsolationPolicy {
    class: "in-process.trusted" | "subprocess.restricted";
    wallTimeMillis?: number;
    cpuTimeMillis?: number;
    memoryBytes?: number;
    maxProcesses?: number;
    maxOutputBytes?: number;
    maxOutputFiles?: number;
    maxProtocolBytes?: number;
  }
  export interface GateBuilder {
    after(job: JobRef): GateBuilder;
  }
  export interface JobBuilder {
    after(job: JobRef): JobBuilder;
    budget(claim: BudgetClaim): JobBuilder;
    isolation(policy: IsolationPolicy): JobBuilder;
  }
  export interface MapBuilder {
    pageSize(value: number): MapBuilder;
    maxItems(value: number): MapBuilder;
    maxMaterializedAhead(value: number): MapBuilder;
    budget(claim: BudgetClaim): MapBuilder;
    isolation(policy: IsolationPolicy): MapBuilder;
  }
  export interface ReduceBuilder {
    fanIn(value: number): ReduceBuilder;
    maxLevels(value: number): ReduceBuilder;
    budget(claim: BudgetClaim): ReduceBuilder;
    isolation(policy: IsolationPolicy): ReduceBuilder;
  }
  export interface PlanBuilder {
    budget(
      account: string,
      options: {limits: BudgetAmounts; policyDigest: string},
    ): PlanBuilder;
    gate<TDecision = unknown>(
      name: string,
      options: {
        schema: string;
        timeoutMs?: number;
        requiredRole: string;
        onReject?: "fail-run" | "cancel-branch";
        onExpire?: "fail-run" | "cancel-branch";
      },
      configure?: (gate: GateBuilder) => void,
    ): ValueRef<TDecision>;
    input<T = unknown>(
      name: string,
      options: {schema: string},
    ): ValueRef<T>;
    inputSet<T = unknown>(
      name: string,
      options: {itemSchema: string; manifestSchema: string; maxItems: number},
    ): SetRef<T>;
    task(
      name: string,
      task: unknown,
      build?: (job: JobBuilder) => void,
    ): JobRef;
    map<I, O>(
      name: string,
      source: SetRef<I>,
      task: (item: ValueRef<I>) => unknown,
      build?: (map: MapBuilder) => void,
    ): SetRef<O>;
    reduce<I, O>(
      name: string,
      source: SetRef<I>,
      task: (partition: ValueRef<readonly I[]>) => unknown,
      build?: (reduce: ReduceBuilder) => void,
    ): ValueRef<O>;
    output(name: string, value: ValueRef): PlanBuilder;
    outputSet(name: string, value: SetRef): PlanBuilder;
  }
  export interface Workflow {}
  export interface WorkflowPlanV3 { readonly schema: "scraper-workflow-plan/v3" }
  export function define(
    name: string,
    build: (plan: PlanBuilder) => void,
  ): Workflow;
  export function toIR(value: Workflow): unknown;
  export function validate(value: Workflow): {ok: boolean; errors: string[]};
  export function digest(value: Workflow): string;
  export function compile(value: Workflow): WorkflowPlanV3;
}
