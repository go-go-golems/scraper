package workflowv3runtime

// TaskTypeScript describes the stable trusted task ABI exposed by
// require("workflow/task") and the lease-scoped context passed to entrypoints.
func TaskTypeScript() string {
	return `declare module "workflow/task" {
  export interface ArtifactInput {
    readonly schema: string;
    readonly digest: string;
    readonly mediaType: string;
    readonly size: number;
    readonly path: string;
  }
  export interface TaskIdentity {
    readonly runId: string;
    readonly nodeKey: string;
    readonly attempt: number;
    readonly operationKey: string;
  }
  export interface OutputRef {
    readonly schema: string;
    readonly digest: string;
    readonly mediaType: string;
    readonly size: number;
    readonly locator: string;
  }
  export interface TaskOutputs {
    putJSON(
      port: string,
      options: {schema: string; value: unknown},
    ): Promise<OutputRef>;
  }
  export interface TaskContext {
    input(): Record<string, ArtifactInput>;
    identity(): TaskIdentity;
    checkpoint(): void;
    readonly outputs: TaskOutputs;
  }
  export interface TaskFailure {
    class: string;
    code: string;
    retryable: boolean;
    message: string;
  }
  export function implementation<T>(
    fn: (context: TaskContext) => T | Promise<T>,
  ): (context: TaskContext) => T | Promise<T>;
  export function success<T>(outputs: T): T;
  export function failure(value: TaskFailure): TaskFailure;
}
`
}
