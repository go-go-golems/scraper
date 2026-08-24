#!/usr/bin/env node

import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import {createRequire} from "node:module";
import {fileURLToPath} from "node:url";

const require = createRequire(import.meta.url);
const here = path.dirname(fileURLToPath(import.meta.url));
const fixture = path.join(here, "fixtures/customer-task-bundle");

function sha256(value) {
  return `sha256:${crypto.createHash("sha256").update(value).digest("hex")}`;
}

function filesUnder(root) {
  const out = [];
  function walk(current) {
    for (const entry of fs.readdirSync(current, {withFileTypes: true}).sort((a, b) => a.name.localeCompare(b.name))) {
      const absolute = path.join(current, entry.name);
      if (entry.isDirectory()) walk(absolute);
      else out.push(absolute);
    }
  }
  walk(root);
  return out;
}

function digestBundle(root) {
  const hash = crypto.createHash("sha256");
  for (const file of filesUnder(root)) {
    const relative = path.relative(root, file).split(path.sep).join("/");
    hash.update(relative);
    hash.update("\0");
    hash.update(fs.readFileSync(file));
    hash.update("\0");
  }
  return `sha256:${hash.digest("hex")}`;
}

class CandidateRegistry {
  constructor(bundleRoot, bundleDigest) {
    this.bundleRoot = bundleRoot;
    this.bundleDigest = bundleDigest;
    this.entries = new Map();
    this.sealed = false;
  }

  task(descriptor) {
    if (this.sealed) throw new Error("registry is sealed");
    const key = `${descriptor.kind}@${descriptor.version}`;
    if (this.entries.has(key)) throw new Error(`duplicate task ${key}`);

    const [modulePath, exportName] = descriptor.entrypoint.split("#");
    const absolute = path.resolve(this.bundleRoot, modulePath);
    const relative = path.relative(this.bundleRoot, absolute);
    if (relative.startsWith("..") || path.isAbsolute(relative)) {
      throw new Error(`entrypoint escapes bundle: ${descriptor.entrypoint}`);
    }
    const implementation = require(absolute);
    if (typeof implementation[exportName] !== "function") {
      throw new Error(`missing export ${exportName} in ${modulePath}`);
    }

    this.entries.set(key, Object.freeze({
      ...descriptor,
      implementation: Object.freeze({
        language: "javascript",
        bundleDigest: this.bundleDigest,
        entrypoint: descriptor.entrypoint,
        abiVersion: "scraper-js-task/v1",
      }),
    }));
  }

  seal() {
    this.sealed = true;
    const tasks = [...this.entries.values()].sort((a, b) =>
      `${a.kind}@${a.version}`.localeCompare(`${b.kind}@${b.version}`));
    const registryDigest = sha256(JSON.stringify(tasks));
    return Object.freeze({registryDigest, tasks: Object.freeze(tasks)});
  }
}

function exactMatch(task, requirement) {
  return task.kind === requirement.kind
    && task.version === requirement.version
    && task.implementation.bundleDigest === requirement.bundleDigest
    && task.implementation.entrypoint === requirement.entrypoint
    && task.implementation.abiVersion === requirement.abiVersion;
}

const bundleDigest = digestBundle(fixture);
const candidate = new CandidateRegistry(fixture, bundleDigest);
const register = require(path.join(fixture, "catalog.cjs"));
register(candidate);
const generation = candidate.seal();

const required = {
  kind: "acme.customer.normalize",
  version: "v1",
  bundleDigest,
  entrypoint: "./tasks/normalize.cjs#run",
  abiVersion: "scraper-js-task/v1",
};
const normalize = generation.tasks.find(task => task.kind === required.kind);

const output = {
  schemaVersion: "js-task-bundle-registration-probe/v1",
  bundle: {
    name: "acme-customer-tasks",
    digest: bundleDigest,
    files: filesUnder(fixture).map(file => path.relative(fixture, file).split(path.sep).join("/")),
  },
  workerGeneration: {
    registryDigest: generation.registryDigest,
    tasks: generation.tasks.map(task => ({
      kind: task.kind,
      version: task.version,
      resource: task.resource,
      implementation: task.implementation,
    })),
  },
  matching: {
    exactRequirementAccepted: exactMatch(normalize, required),
    wrongBundleRejected: !exactMatch(normalize, {...required, bundleDigest: "sha256:wrong"}),
    wrongVersionRejected: !exactMatch(normalize, {...required, version: "v2"}),
    wrongEntrypointRejected: !exactMatch(normalize, {...required, entrypoint: "./tasks/other.cjs#run"}),
  },
};

if (!Object.values(output.matching).every(Boolean)) {
  throw new Error(`matching assertions failed: ${JSON.stringify(output.matching)}`);
}

process.stdout.write(`${JSON.stringify(output, null, 2)}\n`);
