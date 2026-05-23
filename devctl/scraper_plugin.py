#!/usr/bin/env python3
"""devctl plugin for the scraper development stack.

The plugin keeps repo-specific startup knowledge here and lets devctl own
process supervision, logs, status, and shutdown.
"""

from __future__ import annotations

import json
import os
import shutil
import subprocess
import sys
from pathlib import Path
from typing import Any, Dict, List

API_HOST = "127.0.0.1"
API_PORT = 8080
WEB_HOST = "127.0.0.1"
WEB_PORT = 5173
REDIS_HOST = "127.0.0.1"
REDIS_PORT = 6379
REDIS_CONTAINER = "scraper-devctl-redis"


def emit(obj: Dict[str, Any]) -> None:
    sys.stdout.write(json.dumps(obj, separators=(",", ":")) + "\n")
    sys.stdout.flush()


def log(msg: str) -> None:
    sys.stderr.write(msg + "\n")
    sys.stderr.flush()


def repo_root(ctx: Dict[str, Any]) -> Path:
    root = ctx.get("repo_root") or os.getcwd()
    return Path(root).resolve()


def rel(root: Path, *parts: str) -> str:
    return str(root.joinpath(*parts))


def has_executable(name: str) -> bool:
    return shutil.which(name) is not None


def docker_daemon_available() -> bool:
    if not has_executable("docker"):
        return False
    try:
        result = subprocess.run(
            ["docker", "info"],
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            timeout=5,
            check=False,
        )
        return result.returncode == 0
    except Exception:
        return False


def response_ok(request_id: str, output: Dict[str, Any]) -> None:
    emit({"type": "response", "request_id": request_id, "ok": True, "output": output})


def response_error(request_id: str, code: str, message: str) -> None:
    emit({"type": "response", "request_id": request_id, "ok": False, "error": {"code": code, "message": message}})


def config_mutate(ctx: Dict[str, Any]) -> Dict[str, Any]:
    root = repo_root(ctx)
    api_url = f"http://{API_HOST}:{API_PORT}"
    web_url = f"http://{WEB_HOST}:{WEB_PORT}"
    redis_address = f"{REDIS_HOST}:{REDIS_PORT}"
    return {
        "config_patch": {
            "set": {
                "services.redis.port": REDIS_PORT,
                "services.redis.address": redis_address,
                "services.api.port": API_PORT,
                "services.api.url": api_url,
                "services.web.port": WEB_PORT,
                "services.web.url": web_url,
                "env.SCRAPER_DEV_STATE_DIR": rel(root, "state", "devctl"),
                "env.SCRAPER_ENGINE_DB": rel(root, "state", "devctl", "engine.db"),
                "env.SCRAPER_SITES_DIR": rel(root, "state", "devctl", "sites"),
                "env.SCRAPER_RUNTIME_EVENTS_DB": rel(root, "state", "devctl", "runtime-events-sessionstream.db"),
                "env.SCRAPER_EVENTS_BACKEND": "redis",
                "env.SCRAPER_EVENTS_REDIS_ADDRESS": redis_address,
                "env.SCRAPER_API_URL": api_url,
                "env.SCRAPER_WEB_URL": web_url,
            },
            "unset": [],
        }
    }


def validate(ctx: Dict[str, Any]) -> Dict[str, Any]:
    root = repo_root(ctx)
    errors: List[Dict[str, str]] = []
    warnings: List[Dict[str, str]] = []

    checks = {
        "go": "Install Go and ensure `go` is on PATH.",
        "pnpm": "Install pnpm and ensure `pnpm` is on PATH.",
        "docker": "Install Docker or provide a local Redis and adjust the devctl plugin.",
    }
    for exe, hint in checks.items():
        if not has_executable(exe):
            errors.append({"code": "E_MISSING_EXECUTABLE", "message": f"{exe} not found. {hint}"})

    if has_executable("docker") and not docker_daemon_available():
        errors.append({"code": "E_DOCKER_DAEMON", "message": "docker is installed, but the Docker daemon is not reachable."})

    required_paths = [
        root / "cmd" / "scraper",
        root / "sites",
        root / "web" / "package.json",
        root / "go.mod",
    ]
    for path in required_paths:
        if not path.exists():
            errors.append({"code": "E_MISSING_PATH", "message": f"required path is missing: {path}"})

    if not (root / "web" / "node_modules").exists():
        warnings.append({
            "code": "W_NODE_MODULES_MISSING",
            "message": "web/node_modules is missing; run `cd web && pnpm install` before `devctl up`.",
        })

    return {"valid": len(errors) == 0, "errors": errors, "warnings": warnings}


def launch_plan(ctx: Dict[str, Any]) -> Dict[str, Any]:
    root = repo_root(ctx)
    state_dir = rel(root, "state", "devctl")
    engine_db = rel(root, "state", "devctl", "engine.db")
    sites_dir = rel(root, "state", "devctl", "sites")
    runtime_events_db = rel(root, "state", "devctl", "runtime-events-sessionstream.db")
    redis_address = f"{REDIS_HOST}:{REDIS_PORT}"

    common_env = {
        "SCRAPER_DEV_STATE_DIR": state_dir,
        "SCRAPER_ENGINE_DB": engine_db,
        "SCRAPER_SITES_DIR": sites_dir,
        "SCRAPER_RUNTIME_EVENTS_DB": runtime_events_db,
        "SCRAPER_EVENTS_BACKEND": "redis",
        "SCRAPER_EVENTS_REDIS_ADDRESS": redis_address,
    }

    mkdir_state = f"mkdir -p {json.dumps(state_dir)} {json.dumps(sites_dir)}"
    scraper_global = "go run ./cmd/scraper --sites-manifest-dir ./sites --log-level debug"
    event_flags = (
        "--events-backend redis "
        f"--events-redis-address {redis_address} "
        "--events-redis-stream-maxlen 10000"
    )

    return {
        "services": [
            {
                "name": "redis",
                "cwd": ".",
                "command": [
                    "bash",
                    "--noprofile",
                    "--norc",
                    "-lc",
                    "docker rm -f scraper-devctl-redis >/dev/null 2>&1 || true; "
                    "exec docker run --rm --name scraper-devctl-redis "
                    "-p 127.0.0.1:6379:6379 redis:7-alpine",
                ],
                "health": {"type": "tcp", "address": redis_address, "timeout_ms": 60000},
            },
            {
                "name": "api",
                "cwd": ".",
                "command": [
                    "bash",
                    "--noprofile",
                    "--norc",
                    "-lc",
                    f"{mkdir_state}; exec {scraper_global} api serve "
                    f"--address {API_HOST}:{API_PORT} "
                    f"--engine-db {json.dumps(engine_db)} "
                    f"--sites-dir {json.dumps(sites_dir)} "
                    f"--events-sessionstream-db {json.dumps(runtime_events_db)} "
                    f"--events-recent-limit 1000 "
                    f"{event_flags}",
                ],
                "env": common_env,
                "health": {"type": "http", "url": f"http://{API_HOST}:{API_PORT}/healthz", "timeout_ms": 60000},
            },
            {
                "name": "worker",
                "cwd": ".",
                "command": [
                    "bash",
                    "--noprofile",
                    "--norc",
                    "-lc",
                    f"{mkdir_state}; exec {scraper_global} worker run "
                    f"--engine-db {json.dumps(engine_db)} "
                    f"--sites-dir {json.dumps(sites_dir)} "
                    f"--worker-id scraper-devctl-worker "
                    f"--poll-interval 250ms "
                    f"{event_flags}",
                ],
                "env": common_env,
            },
            {
                "name": "web",
                "cwd": "web",
                "command": ["pnpm", "dev", "--host", WEB_HOST, "--port", str(WEB_PORT)],
                "env": {
                    **common_env,
                    "SCRAPER_API_URL": f"http://{API_HOST}:{API_PORT}",
                    "BROWSER": "none",
                },
                "health": {"type": "http", "url": f"http://{WEB_HOST}:{WEB_PORT}/", "timeout_ms": 60000},
            },
        ]
    }


emit({
    "type": "handshake",
    "protocol_version": "v2",
    "plugin_name": "scraper-dev-stack",
    "capabilities": {"ops": ["config.mutate", "validate.run", "launch.plan"]},
    "declares": {"side_effects": "process", "idempotent": False},
})

for raw_line in sys.stdin:
    line = raw_line.strip()
    if not line:
        continue
    try:
        req = json.loads(line)
        request_id = req.get("request_id", "")
        op = req.get("op", "")
        ctx = req.get("ctx", {}) or {}

        if op == "config.mutate":
            response_ok(request_id, config_mutate(ctx))
        elif op == "validate.run":
            response_ok(request_id, validate(ctx))
        elif op == "launch.plan":
            response_ok(request_id, launch_plan(ctx))
        else:
            response_error(request_id, "E_UNSUPPORTED", f"unsupported op: {op}")
    except Exception as exc:  # keep protocol valid even on plugin bugs
        rid = ""
        try:
            rid = json.loads(line).get("request_id", "")
        except Exception:
            pass
        log(f"plugin error: {exc}")
        response_error(rid, "E_PLUGIN", str(exc))
