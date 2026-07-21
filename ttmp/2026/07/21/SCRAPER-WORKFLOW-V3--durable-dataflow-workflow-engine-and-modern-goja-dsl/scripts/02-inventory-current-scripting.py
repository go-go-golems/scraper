#!/usr/bin/env python3
"""Inventory the current scraper scripting surface from source.

The script is intentionally static: it does not execute site scripts or inspect
runtime databases. Its JSON output can be committed as architecture evidence.
"""

from __future__ import annotations

import argparse
import json
import re
from pathlib import Path

CTX_SET = re.compile(r'ctxObj\.Set\("([^"]+)"')
REQUIRE = re.compile(r'require\(["\']([^"\']+)["\']\)')


def fields_of_struct(text: str, name: str) -> list[str]:
    match = re.search(rf"type {re.escape(name)} struct \{{(.*?)\n\}}", text, re.S)
    if not match:
        return []
    return re.findall(r"^\s*([A-Z][A-Za-z0-9_]*)\s+", match.group(1), re.M)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--repo", default=".")
    parser.add_argument("--out", required=True)
    args = parser.parse_args()
    repo = Path(args.repo).resolve()

    execution = (repo / "pkg/js/runtime/executor.go").read_text()
    submission = (repo / "pkg/sites/submitverbs/runtime.go").read_text()
    model = (repo / "pkg/engine/model/types.go").read_text()
    gomod = (repo / "go.mod").read_text()
    version = re.search(r"github.com/go-go-golems/go-go-goja\s+(v\S+)", gomod)

    script_files = sorted((repo / "sites").glob("*/scripts/**/*.js"))
    verb_files = sorted((repo / "sites").glob("*/verbs/**/*.js"))
    requires: dict[str, int] = {}
    for path in script_files + verb_files:
        for name in REQUIRE.findall(path.read_text()):
            requires[name] = requires.get(name, 0) + 1

    report = {
        "schemaVersion": "scraper-scripting-inventory/v1",
        "goGoGojaVersion": version.group(1) if version else "unknown",
        "executionContextPropertiesAndMethods": CTX_SET.findall(execution),
        "submissionContextPropertiesAndMethods": CTX_SET.findall(submission),
        "operationSpecFields": fields_of_struct(model, "OpSpec"),
        "operationResultFields": fields_of_struct(model, "OpResult"),
        "siteScriptFiles": len(script_files),
        "siteVerbFiles": len(verb_files),
        "literalRequireTargets": dict(sorted(requires.items())),
        "observations": [
            "Execution and submission contexts are assembled independently.",
            "The JavaScript API is implemented by raw ctx object property installation rather than a NativeModule contract.",
            "Site source files are CommonJS JavaScript; no TypeScript source appears in the site roots.",
            "Operation specs expose engine storage fields directly to scripts.",
        ],
    }
    Path(args.out).write_text(json.dumps(report, indent=2, sort_keys=True) + "\n")
    print(json.dumps({"out": args.out, "scripts": len(script_files), "verbs": len(verb_files), "requires": len(requires)}, sort_keys=True))


if __name__ == "__main__":
    main()
