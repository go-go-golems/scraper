declare module "workflow" {
  export interface ValueRef<T = unknown> { readonly schema?: string }
  export interface JobRef<T = unknown> {
    output(name: string): ValueRef<T>;
  }
  export interface JobBuilder { after(job: JobRef): JobBuilder }
  export interface PlanBuilder {
    input<T = unknown>(
      name: string,
      options: {schema: string},
    ): ValueRef<T>;
    task(
      name: string,
      task: unknown,
      build?: (job: JobBuilder) => void,
    ): JobRef;
    output(name: string, value: ValueRef): PlanBuilder;
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
