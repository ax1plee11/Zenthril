#!/usr/bin/env python3
"""Bounded local-only WebSocket flooding/rate-limit check.

This script is intentionally capped and refuses non-local targets. It is a
regression check for rate limiting, not a denial-of-service tool.
"""

from __future__ import annotations

import argparse
import json
import sys
import time
from urllib.parse import urlparse, urlencode, urlunparse, parse_qsl

try:
    import websocket  # type: ignore
except ImportError:
    print("Install dependency for local testing: pip install websocket-client", file=sys.stderr)
    sys.exit(2)


LOCAL_HOSTS = {"127.0.0.1", "localhost", "::1"}


def with_ticket(ws_url: str, ticket: str) -> str:
    parsed = urlparse(ws_url)
    query = dict(parse_qsl(parsed.query))
    query["ticket"] = ticket
    return urlunparse(parsed._replace(query=urlencode(query)))


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--ws-url", default="ws://127.0.0.1:8080/ws")
    parser.add_argument("--ticket", required=True)
    parser.add_argument("--origin", default="http://localhost:5173")
    parser.add_argument("--max-messages", type=int, default=130)
    args = parser.parse_args()

    host = urlparse(args.ws_url).hostname
    if host not in LOCAL_HOSTS:
        print("Refusing non-local target. Use this only against local authorized instances.", file=sys.stderr)
        return 2

    count = max(1, min(args.max_messages, 150))
    ws = websocket.create_connection(with_ticket(args.ws_url, args.ticket), origin=args.origin, timeout=5)
    try:
        closed_or_limited = False
        for i in range(count):
            ws.send(json.dumps({"type": "ping", "n": i}))
            ws.settimeout(0.2)
            try:
                msg = ws.recv()
                if "rate_limited" in msg:
                    closed_or_limited = True
                    break
            except Exception:
                closed_or_limited = True
                break
            time.sleep(0.001)
        print(f"sent={i + 1} limited_or_closed={closed_or_limited}")
        return 0 if closed_or_limited else 1
    finally:
        ws.close()


if __name__ == "__main__":
    sys.exit(main())
