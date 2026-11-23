#!/usr/bin/env python3
"""Fake access-log generator for the loglens project.

Usage:
  python3 gen_logs.py 10000 > access.log          # 10k lines
  python3 gen_logs.py 1000000 > big-1.log         # 1M lines
  python3 gen_logs.py 0 --rate 50 >> live.log     # endless, 50 lines/sec
"""

import argparse
import os
import random
import sys
import time
from datetime import datetime, timedelta, timezone

PATHS = [
    "/api/users",
    "/api/orders",
    "/api/login",
    "/api/products",
    "/health",
    "/api/reports/export",
]
METHODS = ["GET"] * 6 + ["POST"] * 3 + ["PUT", "PATCH", "DELETE"]
IPS = [f"10.0.{random.randint(0, 3)}.{random.randint(1, 254)}" for _ in range(50)]
BAD = [
    "",
    "GARBAGE LINE",
    "2026-11-16T10:00:00Z GET /api/users abc 12ms 10.0.0.1",  # bad status
    "2026-11-16T10:00:00Z GET /api/users 200",  # missing fields
    "2026-11-16T10:00:00Z FETCH /api/users 200 5ms 10.0.0.1",  # unknown method
    "2026-11-16T10:00:00Z GET /api/users 200 fast 10.0.0.1",
]  # bad latency


def status():
    r = random.random()
    if r < 0.85:
        return random.choice([200, 200, 200, 201, 204, 304])
    if r < 0.95:
        return random.choice([400, 401, 403, 404, 429])
    return random.choice([500, 502, 503])


def line(ts):
    path = random.choice(PATHS)
    latency = max(1, int(random.lognormvariate(3.5, 0.8)))
    if path == "/api/reports/export":
        latency *= 10  # one slow endpoint
    return (
        f"{ts:%Y-%m-%dT%H:%M:%SZ} {random.choice(METHODS)} {path} "
        f"{status()} {latency}ms {random.choice(IPS)}"
    )


p = argparse.ArgumentParser()
p.add_argument("lines", type=int, help="number of lines (0 = endless)")
p.add_argument(
    "--rate", type=float, default=0, help="lines per second (0 = as fast as possible)"
)
args = p.parse_args()

ts, n = datetime.now(timezone.utc), 0
try:
    while args.lines == 0 or n < args.lines:
        ts += timedelta(milliseconds=random.randint(1, 500))
        print(random.choice(BAD) if random.random() < 0.01 else line(ts))
        n += 1
        if args.rate:
            sys.stdout.flush()
            time.sleep(1 / args.rate)
except (BrokenPipeError, KeyboardInterrupt):
    # stdout closed (e.g. piped into `head`) or Ctrl-C: exit quietly
    os.dup2(os.open(os.devnull, os.O_WRONLY), sys.stdout.fileno())
