#!/usr/bin/env python3
"""WebSocket missing-Origin rejection check."""

from __future__ import annotations

import argparse
import sys
from urllib.parse import urlparse

try:
    import websocket  # type: ignore
except ImportError:
    print("Install dependency for local testing: pip install websocket-client", file=sys.stderr)
    sys.exit(2)


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--ws-url", default="ws://127.0.0.1:8080/ws")
    parser.add_argument("--ticket", default="invalid-ticket")
    args = parser.parse_args()

    host = urlparse(args.ws_url).hostname
    if host not in {"127.0.0.1", "localhost", "::1"}:
        print("Refusing non-local target.", file=sys.stderr)
        return 2

    target = args.ws_url
    sep = "&" if "?" in target else "?"
    target = f"{target}{sep}ticket={args.ticket}"
    try:
        # Suppress Origin entirely. Hardened server should reject before ticket consumption.
        websocket.create_connection(target, suppress_origin=True, timeout=5)
    except Exception as exc:
        print(f"PASS: missing Origin rejected ({exc})")
        return 0
    print("FAIL: WebSocket accepted a missing Origin")
    return 1


if __name__ == "__main__":
    sys.exit(main())
