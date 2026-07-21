#!/usr/bin/env python3
"""Check cookbook bundle catalogs against their visible JS implementations."""

from __future__ import annotations

import argparse
import re
from pathlib import Path

BUNDLE = re.compile(
    r"(?ms)^## Bundle (\d+) — `([^`]+)`\n(.*?)(?=^## Bundle |^# Deep )"
)
JAVASCRIPT = re.compile(r"(?ms)^```js\n(.*?)^```$")
TASK = re.compile(r'\btask\("([A-Za-z0-9]+)"')
HANDLER = re.compile(r"(?m)^  async ([A-Za-z0-9]+)\(ctx\) \{")
REQUIRE = re.compile(r'require\("([^"]+)"\)')
MODULES = re.compile(r"(?ms)^  modules: \[(.*?)^  \],|^  modules: \[([^\n]+)\],")
STRING = re.compile(r'"([^"]+)"')
LOCAL_MODULES = {"cookbook-task-support"}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--doc", required=True, type=Path)
    args = parser.parse_args()

    text = args.doc.read_text(encoding="utf-8")
    bundles = list(BUNDLE.finditer(text))
    errors: list[str] = []

    for match in bundles:
        number, name, section = match.groups()
        blocks = JAVASCRIPT.findall(section)
        if len(blocks) < 2:
            errors.append(f"bundle {number} {name}: expected catalog and execution")
            continue

        catalog, execution = blocks[0], blocks[1]
        catalog_tasks = set(TASK.findall(catalog))
        handlers = set(HANDLER.findall(execution))
        if catalog_tasks != handlers:
            errors.append(
                f"bundle {number} {name}: task/handler mismatch "
                f"missing={sorted(catalog_tasks - handlers)} "
                f"extra={sorted(handlers - catalog_tasks)}"
            )

        module_match = MODULES.search(catalog)
        declared = set()
        if module_match:
            declared = set(STRING.findall(next(
                group for group in module_match.groups() if group is not None
            )))
        imported = set(REQUIRE.findall(execution)) - LOCAL_MODULES
        if declared != imported:
            errors.append(
                f"bundle {number} {name}: module mismatch "
                f"undeclared={sorted(imported - declared)} "
                f"unused={sorted(declared - imported)}"
            )

    if len(bundles) != 15:
        errors.append(f"expected 15 bundles, found {len(bundles)}")

    if errors:
        print("\n".join(errors))
        return 1

    print("bundles=15 catalogsMatchHandlers=true modulesMatchImports=true")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
