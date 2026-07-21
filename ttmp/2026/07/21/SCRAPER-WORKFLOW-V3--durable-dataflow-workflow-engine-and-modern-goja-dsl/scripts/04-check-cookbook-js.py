#!/usr/bin/env python3
"""Extract JavaScript fences from the workflow-v3 cookbook and syntax-check them."""

from __future__ import annotations

import argparse
import json
import subprocess
import tempfile
from pathlib import Path


def extract_js_blocks(markdown: str) -> list[tuple[int, str]]:
    blocks: list[tuple[int, str]] = []
    lines = markdown.splitlines()
    in_js = False
    start_line = 0
    current: list[str] = []

    for number, line in enumerate(lines, start=1):
        if not in_js and line.strip() in {"```js", "```javascript"}:
            in_js = True
            start_line = number + 1
            current = []
            continue
        if in_js and line.strip() == "```":
            blocks.append((start_line, "\n".join(current) + "\n"))
            in_js = False
            current = []
            continue
        if in_js:
            current.append(line)

    if in_js:
        raise ValueError(f"unterminated JavaScript fence starting at line {start_line - 1}")
    return blocks


def check_block(index: int, start_line: int, source: str) -> dict[str, object]:
    with tempfile.NamedTemporaryFile(
        mode="w", suffix=f"-cookbook-{index:02d}.js", encoding="utf-8", delete=False
    ) as handle:
        handle.write(source)
        path = Path(handle.name)

    try:
        process = subprocess.run(
            ["node", "--check", str(path)],
            check=False,
            capture_output=True,
            text=True,
        )
    finally:
        path.unlink(missing_ok=True)

    return {
        "index": index,
        "startLine": start_line,
        "ok": process.returncode == 0,
        "error": process.stderr.strip() or None,
    }


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--doc", required=True, type=Path)
    parser.add_argument("--out", type=Path)
    args = parser.parse_args()

    blocks = extract_js_blocks(args.doc.read_text(encoding="utf-8"))
    checks = [check_block(index, line, source) for index, (line, source) in enumerate(blocks, start=1)]
    result = {
        "schemaVersion": "workflow-cookbook-js-check/v1",
        "document": str(args.doc),
        "blockCount": len(checks),
        "passed": sum(1 for check in checks if check["ok"]),
        "failed": sum(1 for check in checks if not check["ok"]),
        "checks": checks,
    }
    rendered = json.dumps(result, indent=2, sort_keys=True) + "\n"
    if args.out:
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(rendered, encoding="utf-8")
    else:
        print(rendered, end="")
    return 1 if result["failed"] else 0


if __name__ == "__main__":
    raise SystemExit(main())
