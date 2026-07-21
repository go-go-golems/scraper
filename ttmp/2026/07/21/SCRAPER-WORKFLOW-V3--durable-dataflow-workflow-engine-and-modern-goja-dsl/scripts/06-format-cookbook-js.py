#!/usr/bin/env python3
"""Format every JavaScript fence in the workflow-v3 cookbook with Deno."""

from __future__ import annotations

import argparse
import re
import subprocess
from pathlib import Path

FENCE = re.compile(r"(?ms)^```js\n(.*?)^```$")


def format_javascript(source: str) -> str:
    result = subprocess.run(
        ["deno", "fmt", "--line-width", "80", "-"],
        input=source,
        text=True,
        capture_output=True,
        check=True,
    )
    return result.stdout.rstrip("\n")


def transform(document: str) -> tuple[str, int]:
    count = 0

    def replace(match: re.Match[str]) -> str:
        nonlocal count
        count += 1
        return "```js\n" + format_javascript(match.group(1)) + "\n```"

    return FENCE.sub(replace, document), count


def long_lines(document: str) -> list[tuple[int, int, str]]:
    findings: list[tuple[int, int, str]] = []
    for match in FENCE.finditer(document):
        first_line = document.count("\n", 0, match.start(1)) + 1
        for offset, line in enumerate(match.group(1).splitlines()):
            if len(line) > 80:
                findings.append((first_line + offset, len(line), line))
    return findings


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--doc", required=True, type=Path)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()

    original = args.doc.read_text(encoding="utf-8")
    formatted, count = transform(original)

    if args.check and formatted != original:
        print(f"{args.doc}: JavaScript fences require formatting")
        return 1

    if not args.check:
        args.doc.write_text(formatted, encoding="utf-8")

    findings = long_lines(formatted)
    if findings:
        for line, width, text in findings:
            print(f"{args.doc}:{line}: width {width}: {text}")
        return 1

    print(f"formatted={count} maxWidth=80 violations=0")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
