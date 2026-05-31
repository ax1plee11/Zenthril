#!/usr/bin/env python3
"""Controlled JWT alg=none rejection check for local Zenthril instances."""

from __future__ import annotations

import argparse
import base64
import json
import sys
import time
import urllib.request
import urllib.error


def b64url(data: bytes) -> str:
    return base64.urlsafe_b64encode(data).decode().rstrip("=")


def make_alg_none_token(user_id: str) -> str:
    header = {"alg": "none", "typ": "JWT"}
    payload = {
        "user_id": user_id,
        "token_type": "access",
        "exp": int(time.time()) + 3600,
    }
    return f"{b64url(json.dumps(header, separators=(',', ':')).encode())}.{b64url(json.dumps(payload, separators=(',', ':')).encode())}."


def request_me(base_url: str, token: str) -> int:
    req = urllib.request.Request(
        f"{base_url.rstrip('/')}/api/v1/auth/me",
        headers={"Authorization": f"Bearer {token}"},
        method="GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=5) as res:
            return res.status
    except urllib.error.HTTPError as exc:
        return exc.code


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base-url", default="http://127.0.0.1:8080")
    parser.add_argument("--user-id", default="00000000-0000-0000-0000-000000000000")
    args = parser.parse_args()

    token = make_alg_none_token(args.user_id)
    status = request_me(args.base_url, token)
    print(f"status={status}")
    if status in (401, 403):
        print("PASS: alg=none token rejected")
        return 0
    print("FAIL: alg=none token was not rejected")
    return 1


if __name__ == "__main__":
    sys.exit(main())
