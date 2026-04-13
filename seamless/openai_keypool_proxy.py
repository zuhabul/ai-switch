#!/usr/bin/env python3
"""
OpenAI-compatible local proxy with key-pool failover.

Purpose:
- Keep one stable base URL for clients (e.g., Codex CLI).
- Retry upstream requests across multiple API keys when 429/5xx occur.
- Avoid client-side process restarts for rate-limit failover.
"""

from __future__ import annotations

import argparse
import asyncio
import os
import threading
import tomllib
from dataclasses import dataclass
from pathlib import Path
from typing import Iterable

import httpx
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse, Response, StreamingResponse
from starlette.background import BackgroundTask

HOP_BY_HOP_HEADERS = {
    "connection",
    "keep-alive",
    "proxy-authenticate",
    "proxy-authorization",
    "te",
    "trailers",
    "transfer-encoding",
    "upgrade",
}

DROP_RESPONSE_HEADERS = HOP_BY_HOP_HEADERS | {"content-length", "server", "date"}
DROP_REQUEST_HEADERS = HOP_BY_HOP_HEADERS | {"host", "authorization", "content-length"}


@dataclass(frozen=True)
class ProxyConfig:
    listen_host: str
    listen_port: int
    upstream_base_url: str
    client_api_key: str | None
    request_timeout_seconds: float
    max_attempts: int
    retry_statuses: tuple[int, ...]
    keys_file: Path


class KeyPool:
    def __init__(self, keys: Iterable[str]) -> None:
        self._keys = [k.strip() for k in keys if k.strip()]
        if not self._keys:
            raise ValueError("No upstream API keys configured.")
        self._lock = threading.Lock()
        self._index = 0
        self._last_used = -1

    @property
    def size(self) -> int:
        return len(self._keys)

    @property
    def last_used_index(self) -> int:
        return self._last_used

    def next_key(self) -> tuple[int, str]:
        with self._lock:
            idx = self._index
            self._index = (self._index + 1) % len(self._keys)
            self._last_used = idx
            return idx, self._keys[idx]


def _expand_path(value: str) -> Path:
    return Path(os.path.expandvars(os.path.expanduser(value))).resolve()


def load_config(path: Path) -> ProxyConfig:
    data = tomllib.loads(path.read_text(encoding="utf-8"))

    listen_host = str(data.get("listen_host", "127.0.0.1"))
    listen_port = int(data.get("listen_port", 8788))
    upstream_base_url = str(data.get("upstream_base_url", "https://api.openai.com/v1")).rstrip("/")
    client_api_key = data.get("client_api_key")
    request_timeout_seconds = float(data.get("request_timeout_seconds", 600))
    max_attempts = int(data.get("max_attempts", 0))
    retry_statuses = tuple(int(v) for v in data.get("retry_statuses", [429, 500, 502, 503, 504]))
    keys_file = _expand_path(str(data.get("keys_file", "~/.config/ai-switch/openai-keypool.keys")))

    if max_attempts <= 0:
        max_attempts = 0  # derive from key count later

    return ProxyConfig(
        listen_host=listen_host,
        listen_port=listen_port,
        upstream_base_url=upstream_base_url,
        client_api_key=str(client_api_key) if client_api_key else None,
        request_timeout_seconds=request_timeout_seconds,
        max_attempts=max_attempts,
        retry_statuses=retry_statuses,
        keys_file=keys_file,
    )


def load_keys(path: Path) -> list[str]:
    if not path.exists():
        raise FileNotFoundError(f"Keys file not found: {path}")
    keys: list[str] = []
    for raw in path.read_text(encoding="utf-8").splitlines():
        line = raw.strip()
        if not line or line.startswith("#"):
            continue
        keys.append(line)
    if not keys:
        raise ValueError(f"No keys found in {path}")
    return keys


def normalize_response_headers(headers: httpx.Headers) -> dict[str, str]:
    out: dict[str, str] = {}
    for k, v in headers.items():
        lk = k.lower()
        if lk in DROP_RESPONSE_HEADERS:
            continue
        out[k] = v
    return out


def forwarded_request_headers(request: Request, upstream_key: str) -> dict[str, str]:
    out: dict[str, str] = {}
    for k, v in request.headers.items():
        lk = k.lower()
        if lk in DROP_REQUEST_HEADERS:
            continue
        out[k] = v
    out["Authorization"] = f"Bearer {upstream_key}"
    return out


def build_app(config: ProxyConfig, key_pool: KeyPool) -> FastAPI:
    timeout = httpx.Timeout(config.request_timeout_seconds)
    client = httpx.AsyncClient(timeout=timeout, follow_redirects=False)

    app = FastAPI(title="OpenAI Keypool Proxy", version="1.0.0")

    @app.on_event("shutdown")
    async def _shutdown_client() -> None:
        await client.aclose()

    @app.get("/health")
    async def health() -> dict[str, object]:
        return {
            "ok": True,
            "keys_total": key_pool.size,
            "last_used_index": key_pool.last_used_index,
            "retry_statuses": list(config.retry_statuses),
            "max_attempts": config.max_attempts or key_pool.size,
            "upstream_base_url": config.upstream_base_url,
        }

    @app.api_route("/v1/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"])
    async def proxy_v1(path: str, request: Request) -> Response:
        if config.client_api_key:
            auth = request.headers.get("authorization", "")
            if auth != f"Bearer {config.client_api_key}":
                raise HTTPException(status_code=401, detail="Invalid proxy API key.")

        body = await request.body()
        target_url = f"{config.upstream_base_url}/{path}"
        attempts = config.max_attempts or key_pool.size
        attempts = max(1, min(attempts, key_pool.size))
        last_status = 0
        last_error: str | None = None

        for attempt in range(attempts):
            _, upstream_key = key_pool.next_key()
            req_headers = forwarded_request_headers(request, upstream_key)
            req = client.build_request(
                method=request.method,
                url=target_url,
                params=request.query_params,
                headers=req_headers,
                content=body if body else None,
            )
            try:
                upstream = await client.send(req, stream=True)
            except Exception as exc:  # network/transient client errors
                last_error = str(exc)
                if attempt + 1 < attempts:
                    await asyncio.sleep(0.05)
                    continue
                return JSONResponse(
                    status_code=502,
                    content={"error": {"message": f"Upstream request failed: {last_error}", "type": "proxy_error"}},
                )

            last_status = upstream.status_code
            if upstream.status_code in config.retry_statuses and attempt + 1 < attempts:
                await upstream.aread()
                await upstream.aclose()
                continue

            resp_headers = normalize_response_headers(upstream.headers)
            content_type = upstream.headers.get("content-type", "").lower()

            if "text/event-stream" in content_type:
                return StreamingResponse(
                    upstream.aiter_bytes(),
                    status_code=upstream.status_code,
                    headers=resp_headers,
                    background=BackgroundTask(upstream.aclose),
                )

            payload = await upstream.aread()
            await upstream.aclose()
            return Response(content=payload, status_code=upstream.status_code, headers=resp_headers)

        return JSONResponse(
            status_code=429 if last_status == 429 else 502,
            content={
                "error": {
                    "message": "All upstream keys exhausted after retry attempts.",
                    "type": "proxy_upstream_exhausted",
                    "last_status": last_status,
                    "last_error": last_error,
                }
            },
        )

    return app


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="OpenAI key-pool proxy for seamless Codex failover")
    parser.add_argument(
        "--config",
        default="~/.config/ai-switch/seamless-proxy.toml",
        help="Path to TOML config file",
    )
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    config_path = _expand_path(args.config)
    if not config_path.exists():
        raise SystemExit(f"Config file not found: {config_path}")

    config = load_config(config_path)
    keys = load_keys(config.keys_file)
    key_pool = KeyPool(keys)
    app = build_app(config, key_pool)

    import uvicorn

    uvicorn.run(
        app,
        host=config.listen_host,
        port=config.listen_port,
        log_level="info",
    )


if __name__ == "__main__":
    main()

