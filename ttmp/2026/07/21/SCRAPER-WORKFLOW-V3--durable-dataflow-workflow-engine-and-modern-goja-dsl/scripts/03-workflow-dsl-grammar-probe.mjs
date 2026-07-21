#!/usr/bin/env node
// Grammar probe only. Production implementation must be Go-backed and exposed
// as require("workflow") through xgoja/v2. Immediate configurator callbacks are
// evaluated here and never appear in the serialized plan.

import { createHash } from "node:crypto";

const refSymbol = Symbol("workflow.ref");
const taskSymbol = Symbol("workflow.task");

function stable(value) {
  if (Array.isArray(value)) return value.map(stable);
  if (value && typeof value === "object") {
    return Object.fromEntries(Object.keys(value).sort().map((key) => [key, stable(value[key])]));
  }
  return value;
}

function digest(value) {
  return `sha256:${createHash("sha256").update(JSON.stringify(stable(value))).digest("hex")}`;
}

function reference(kind, id) {
  return Object.freeze({ [refSymbol]: { kind, id } });
}

function ref(value, expected) {
  const hidden = value?.[refSymbol];
  if (!hidden || (expected && hidden.kind !== expected)) throw new TypeError(`expected ${expected || "workflow"} reference`);
  return hidden;
}

function taskDescriptor(kind, config = {}) {
  return Object.freeze({ [taskSymbol]: { kind, version: "v1", config: stable(config) } });
}

function task(value) {
  const hidden = value?.[taskSymbol];
  if (!hidden) throw new TypeError("expected task descriptor");
  return hidden;
}

function configure(builder, callback) {
  if (callback !== undefined) {
    if (typeof callback !== "function") throw new TypeError("configurator must be a function");
    callback(builder);
  }
  return builder;
}

function jobBuilder(target) {
  const api = {
    resource(name) { target.resource = name; return api; },
    timeout(value) { target.timeout = value; return api; },
    retry(options) { target.retry = stable(options); return api; },
    priority(value) { target.priority = value; return api; },
  };
  return api;
}

function resourceBuilder(target) {
  const api = {
    maxInFlight(value) { target.maxInFlight = value; return api; },
    rate(options) { target.rate = stable(options); return api; },
    fairness(value) { target.fairness = value; return api; },
  };
  return api;
}

function plan(name, callback) {
  const spec = { schemaVersion: "scraper-workflow-plan/v3", name, inputs: [], resources: [], jobs: [], outputs: [] };
  const p = {
    input(id, options) {
      spec.inputs.push({ id, ...stable(options) });
      return reference("value", id);
    },
    resource(name, cb) {
      const value = { name };
      configure(resourceBuilder(value), cb);
      spec.resources.push(value);
      return p;
    },
    map(id, source, descriptor, cb) {
      const sourceRef = ref(source, "value");
      const value = { id, mode: "map", from: sourceRef.id, task: task(descriptor) };
      configure(jobBuilder(value), cb);
      spec.jobs.push(value);
      return reference("value", id);
    },
    reduce(id, source, descriptor, cb) {
      const sourceRef = ref(source, "value");
      const value = { id, mode: "reduce", from: sourceRef.id, task: task(descriptor) };
      configure(jobBuilder(value), cb);
      spec.jobs.push(value);
      return reference("value", id);
    },
    output(name, value) {
      spec.outputs.push({ name, from: ref(value, "value").id });
      return p;
    },
  };
  configure(p, callback);
  const ids = new Set(spec.inputs.map((x) => x.id));
  for (const job of spec.jobs) {
    if (ids.has(job.id)) throw new Error(`duplicate workflow value ${job.id}`);
    if (!ids.has(job.from)) throw new Error(`unresolved workflow value ${job.from}`);
    if (!spec.resources.some((r) => r.name === job.resource)) throw new Error(`unknown resource ${job.resource}`);
    ids.add(job.id);
  }
  for (const output of spec.outputs) if (!ids.has(output.from)) throw new Error(`unresolved output ${output.from}`);
  const normalized = stable(spec);
  return Object.freeze({
    validate: () => ({ ok: true, issues: [] }),
    explain: () => ({ jobs: spec.jobs.length, resources: spec.resources.map((r) => r.name), outputs: spec.outputs }),
    toPlan: () => ({ ...normalized, digest: digest(normalized) }),
  });
}

const workflow = { plan, tasks: { native: taskDescriptor } };
const rag = {
  tasks: {
    combinedSummaryQuestions: (options) => taskDescriptor("rag.combined-summary-questions", options),
    embedRepresentations: (options) => taskDescriptor("rag.embed-representations", options),
    publishPreparedCorpus: (options) => taskDescriptor("rag.publish-prepared-corpus", options),
  },
};

// Target authoring shape for a researchctl/RAG script importing both modules.
const preparation = workflow.plan("ttc-preparation", (p) => {
  p.resource("llm.generate", (r) => r.maxInFlight(3).rate({ requestsPerMinute: 30 }).fairness("fifo"));
  p.resource("embedding.local", (r) => r.maxInFlight(1).fairness("fifo"));
  p.resource("local.publish", (r) => r.maxInFlight(1));

  const chunks = p.input("chunks", {
    kind: "artifact-set",
    schema: "rag-chunk-ref-set/v1",
    role: "corpus-chunks",
  });
  const generated = p.map(
    "generated",
    chunks,
    rag.tasks.combinedSummaryQuestions({ model: "generator", questionsPerChunk: 4 }),
    (job) => job.resource("llm.generate").timeout("3m").retry({ maxAttempts: 3, class: "provider-response" }),
  );
  const embedded = p.map(
    "embedded",
    generated,
    rag.tasks.embedRepresentations({ model: "embedding", batchSize: 16 }),
    (job) => job.resource("embedding.local").timeout("3m").retry({ maxAttempts: 3, class: "provider" }),
  );
  const published = p.reduce(
    "published",
    embedded,
    rag.tasks.publishPreparedCorpus({ shardSize: 128, reopen: true }),
    (job) => job.resource("local.publish").timeout("10m").retry({ maxAttempts: 1, class: "permanent" }),
  );
  p.output("preparedCorpus", published);
});

const output = preparation.toPlan();
if (JSON.stringify(output).includes("function")) throw new Error("serialized plan retained a callback");
process.stdout.write(`${JSON.stringify({ plan: output, explanation: preparation.explain() }, null, 2)}\n`);
