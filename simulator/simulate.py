#!/usr/bin/env python3
"""Feeds the leaderboard with fake traffic over gRPC.

1. Registers N players through the REST API (the gRPC API refuses unknown players).
2. Streams random scores to the Go backend's gRPC Ingest service.

Open the web UI while it runs to watch the leaderboard move live.
"""
import argparse
import json
import os
import random
import signal
import sys
import time
import urllib.error
import urllib.request

import grpc

sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "gen"))
from leaderboard.v1 import leaderboard_pb2 as pb  # noqa: E402
from leaderboard.v1 import leaderboard_pb2_grpc as pb_grpc  # noqa: E402

PASSWORD = "simulator-pass"  # fake accounts; only meant for local demos
GAME_SCALE = {"chess": 2000, "pong": 500, "tetris": 80000, "snake": 1200, "racing": 15000}


def register_players(rest: str, names: list[str]) -> None:
    for name in names:
        req = urllib.request.Request(
            f"{rest}/auth/register",
            data=json.dumps({"username": name, "password": PASSWORD}).encode(),
            headers={"Content-Type": "application/json"},
            method="POST",
        )
        try:
            urllib.request.urlopen(req, timeout=5).close()
        except urllib.error.HTTPError as e:
            if e.code != 409:  # 409 = already registered on an earlier run
                raise SystemExit(f"registering {name} failed: HTTP {e.code} {e.read().decode()}")
        except urllib.error.URLError as e:
            raise SystemExit(f"cannot reach REST API at {rest}: {e.reason}")


class Feeder:
    """Generates plausible scores: each player has a skill; scores vary around it."""

    def __init__(self, players: list[str], games: list[str], rng: random.Random):
        self.rng = rng
        self.players = players
        self.games = games
        self.skill = {p: rng.uniform(0.3, 1.0) for p in players}

    def next(self) -> pb.SubmitScoreRequest:
        player = self.rng.choice(self.players)
        game = self.rng.choice(self.games)
        scale = GAME_SCALE.get(game, 1000)
        score = int(scale * self.skill[player] * self.rng.uniform(0.4, 1.15))
        return pb.SubmitScoreRequest(username=player, game=game, score=score)


def run(args) -> int:
    rng = random.Random(args.seed)
    names = [f"sim_{i:02d}" for i in range(1, args.players + 1)]
    games = [g.strip() for g in args.games.split(",") if g.strip()]
    register_players(args.rest, names)
    print(f"registered {len(names)} players; feeding {args.grpc} at ~{args.rate:g} scores/s", flush=True)

    feeder = Feeder(names, games, rng)
    metadata = (("x-api-key", args.api_key),)
    deadline = time.monotonic() + args.duration if args.duration else None
    stopping = False

    def stop(*_):
        nonlocal stopping
        stopping = True

    signal.signal(signal.SIGINT, stop)
    signal.signal(signal.SIGTERM, stop)

    def paced():
        """Yields requests at the target rate until duration elapses or we are interrupted."""
        interval = 1.0 / args.rate
        while not stopping and (deadline is None or time.monotonic() < deadline):
            req = feeder.next()
            if args.verbose:
                print(f"  {req.username:8} {req.game:8} {req.score}", flush=True)
            yield req
            time.sleep(interval)

    with grpc.insecure_channel(args.grpc) as channel:
        client = pb_grpc.IngestStub(channel)
        try:
            if args.unary:
                ok = bad = 0
                for req in paced():
                    try:
                        res = client.SubmitScore(req, metadata=metadata)
                        ok += 1
                        if res.new_personal_best and args.verbose:
                            print(f"    new best! rank #{res.game_rank} in {req.game}", flush=True)
                    except grpc.RpcError as e:
                        if e.code() == grpc.StatusCode.UNAUTHENTICATED:
                            raise SystemExit(f"gRPC auth failed: {e.details()}")
                        bad += 1
                print(f"done: accepted={ok} rejected={bad}")
            else:
                summary = client.StreamScores(paced(), metadata=metadata)
                print(f"done: accepted={summary.accepted} rejected={summary.rejected}")
        except grpc.RpcError as e:
            if e.code() == grpc.StatusCode.UNAUTHENTICATED:
                raise SystemExit(f"gRPC auth failed: {e.details()} (check --api-key / INGEST_API_KEY)")
            if e.code() == grpc.StatusCode.UNAVAILABLE:
                raise SystemExit(f"cannot reach gRPC server at {args.grpc}: {e.details()}")
            raise
    return 0


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--rest", default="http://localhost:8080", help="REST base URL, used only to register players")
    p.add_argument("--grpc", default="localhost:9090", help="gRPC ingest address")
    p.add_argument("--api-key", default=os.environ.get("INGEST_API_KEY", ""), help="defaults to $INGEST_API_KEY")
    p.add_argument("--players", type=int, default=20)
    p.add_argument("--games", default="chess,pong,tetris", help=f"comma list; tuned for: {', '.join(GAME_SCALE)}")
    p.add_argument("--rate", type=float, default=5.0, help="scores per second")
    p.add_argument("--duration", type=float, default=0, help="seconds to run (0 = until Ctrl-C)")
    p.add_argument("--unary", action="store_true", help="one SubmitScore call per score instead of one stream")
    p.add_argument("--seed", type=int, default=None, help="RNG seed for reproducible runs")
    p.add_argument("-v", "--verbose", action="store_true")
    args = p.parse_args()
    if not args.api_key:
        p.error("an API key is required (--api-key or INGEST_API_KEY)")
    if args.rate <= 0 or args.players < 1:
        p.error("--rate and --players must be positive")
    return run(args)


if __name__ == "__main__":
    sys.exit(main())
