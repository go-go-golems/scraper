#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT="$ROOT/sources"
mkdir -p "$OUT"

fetch() {
  local name="$1"
  local url="$2"
  local tmp
  tmp="$(mktemp)"
  trap 'rm -f "$tmp"' RETURN
  defuddle parse "$url" --md >"$tmp"
  {
    printf '<!-- Source: %s -->\n' "$url"
    printf '<!-- Retrieved: 2026-07-21 -->\n\n'
    cat "$tmp"
  } >"$OUT/$name"
  rm -f "$tmp"
  trap - RETURN
}

fetch 01-scraper-project-map.md \
  'https://parc.yolo.scapegoat.dev/note/research/kb/projects/scraper'
fetch 02-go-go-goja-project-map.md \
  'https://parc.yolo.scapegoat.dev/note/research/kb/projects/go-go-goja'
fetch 03-widget-dsl-project-map.md \
  'https://parc.yolo.scapegoat.dev/note/research/kb/projects/widget-dsl'
fetch 04-scraper-workflow-api.md \
  'https://parc.yolo.scapegoat.dev/note/projects/2026/05/25/article-scraper-workflow-api-building-a-public-reusable-durable-workflow-runtime'
fetch 05-goja-fluent-builder-dsls.md \
  'https://parc.yolo.scapegoat.dev/note/projects/2026/07/05/article-goja-fluent-builder-dsls-designing-typed-composable-grammars-in-go-for-javascript'
fetch 06-designing-dsls-with-go-go-goja.md \
  'https://parc.yolo.scapegoat.dev/note/projects/2026/06/22/article-designing-dsls-with-go-go-goja-go-backed-javascript-apis'
fetch 07-data-only-vs-host-access-module-split.md \
  'https://parc.yolo.scapegoat.dev/note/research/kb/tribal/data-only-vs-host-access-module-split'
fetch 08-dsl-normalized-config-compiled-plan.md \
  'https://parc.yolo.scapegoat.dev/note/research/kb/tribal/dsl-normalized-config-compiled-plan'

xgoja help xgoja-v2-reference >"$OUT/09-xgoja-v2-reference.txt"
xgoja help provider-runtime-config-and-host-services >"$OUT/10-xgoja-provider-runtime-config-and-host-services.txt"

# Terminal-oriented help output may be padded with spaces. Normalize every
# captured source so regenerated ticket artifacts remain commit-clean.
python3 - "$OUT" <<'PY'
from pathlib import Path
import sys

for path in sorted(Path(sys.argv[1]).iterdir()):
    if not path.is_file():
        continue
    lines = path.read_text(encoding="utf-8").splitlines()
    path.write_text("\n".join(line.rstrip(" \t") for line in lines).rstrip("\n") + "\n", encoding="utf-8")
PY

printf 'Saved %s source documents under %s\n' "$(find "$OUT" -maxdepth 1 -type f | wc -l)" "$OUT"
