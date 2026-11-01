#!/usr/bin/env python3
"""Short matched tests of compatible peer scheduling; no public torrent peers."""
import argparse
import gc
import hashlib
import json
from pathlib import Path
import statistics

from high_bitrate_check import MIB, run, calibrate
from controlled_peer import LocalSwarm, PeerPlan

POLICIES = [
    ("capacity", {"capacity_aware": True}),
    ("frontier", {"capacity_aware": True, "frontier_recovery": True}),
    ("requests", {"capacity_aware": True, "adaptive_peer_requests": True}),
    ("combined", {"capacity_aware": True, "frontier_recovery": True, "adaptive_peer_requests": True}),
]
CASES = {
    "healthy": ("healthy", 512, 0, 4),
    "stragglers": ("stragglers", 512, 0, 8),
    "small-queue": ("healthy", 32, 250, 1),
    "mixed-peers": ("mixed-peers", 64, 150, 4),
    "outages": ("outages", 64, 100, 4),
    "high-latency": ("healthy", 512, 2500, 4),
    "large-pieces": ("stragglers", 128, 150, 16),
}


def summarize(runs):
    result = {}
    for name, _ in POLICIES:
        rows = [r for r in runs if r["policy"] == name]
        infrastructure = [r for r in rows if r.get('failure_stage') == 'fixture_port_allocation']
        rows = [r for r in rows if r not in infrastructure]
        summary = {"passed": sum(bool(r["passed"]) for r in rows), "total": len(rows),
                   "allocation_failures": len(infrastructure)}
        for key in ("preload_ready_ms", "peak_observed_rss_bytes", "native_redundant_bytes",
                    "cpu_seconds", "cold_seek_ttfb_p95_ms"):
            values = [r[key] for r in rows if r.get(key) is not None]
            if values:
                summary[key] = {"values": values, "median": statistics.median(values)}
        for key in ("blocked_over_250ms_ms", "read_wait_max_ms", "behind_schedule_ms"):
            values = [r["delivery"][key] for r in rows if r.get("delivery")]
            if values:
                summary[key] = {"values": values, "median": statistics.median(values)}
        result[name] = summary
    return result


def accepted_trial(case, row):
    if row['passed']:
        return True
    # The stock two-second urgent horizon cannot fill a pipeline whose every
    # block has 2.5 seconds of service latency. Retain those control timeouts as
    # measured failures, while requiring both adaptive candidates to complete.
    # No missing peers, corruption, capacity breach or candidate failure is waived.
    peers = (row.get('swarm') or {}).get('peers') or []
    return (case == 'high-latency' and row['policy'] in ('capacity', 'frontier')
            and row.get('failure_stage') == 'preload'
            and row.get('failure_type') == 'TimeoutError'
            and row.get('policy_error') is None and bool(peers)
            and sum(p.get('sent_bytes', 0) for p in peers) > 0
            and all(not p.get('capacity_overflows')
                    and p.get('peak_pending_requests', 0) <= 512 for p in peers))


def check(args):
    args.output.mkdir(parents=True, exist_ok=False)
    case, reqq, latency, piece = CASES[args.case]
    fixture = args.output / "owned.bin"
    # Keeps cold seek ranges outside the resident cache, without downloading a
    # movie or introducing FFmpeg/decoder behavior into this scheduling test.
    with fixture.open("xb") as stream:
        # Four disjoint cold seeks must remain possible after read-ahead and
        # earlier seeks. Twelve 16 MiB pieces could exhaust the proof targets.
        for number in range(max(192, 24*piece)):
            stream.write(hashlib.shake_256(number.to_bytes(4, "big")).digest(MIB))
    with args.executable.open("rb") as source:
        digest = hashlib.file_digest(source, "sha256").hexdigest()
    # Exercise the new policies at high bitrate as well as under constrained
    # small-queue/latency fixtures. The latter deliberately use lower demand
    # so insufficient fixture supply cannot masquerade as a policy regression.
    rate = (2*MIB if args.case in ("high-latency", "large-pieces")
            else 12*MIB if args.case == "healthy" else MIB if latency else 4*MIB)
    with LocalSwarm([fixture], peers=[PeerPlan(rate=2*rate)],
                    piece_length=piece*MIB, high_throughput=True) as swarm:
        supply = calibrate(swarm)
    del swarm
    gc.collect()
    if supply < 1.3*rate*8/1_000_000:
        raise RuntimeError("Fixture supply cannot sustain a fair comparison")
    report = {"executable_sha256": digest, "case": args.case, "disk": args.disk,
              "rotations": args.rotations, "promoted": False,
              "baseline": "Legacy with existing capacity guard; each candidate shares that prerequisite",
              "peer_capacity_mbps": supply,
              "synthetic_transport_only": True, "runs": [], "summaries": {}}
    for rotation in range(args.rotations):
        order = POLICIES[rotation % len(POLICIES):] + POLICIES[:rotation % len(POLICIES)]
        if rotation % 2:
            order = list(reversed(order))
        for name, options in order:
            print(f"{args.case} / {rotation + 1} / {name}", flush=True)
            row = run(args.executable, args.output / f"{rotation}-{name}", fixture,
                      "legacy", case, rate, 35 if case == "outages" else 12, 64, True,
                      reqq=reqq, strict=True, latency_ms=latency, piece_mb=piece,
                      startup_mb=4, disk=args.disk, **options)
            row.update(policy=name, rotation=rotation)
            controls = [s.get("stream_control") for s in
                        row.get('preload_samples', []) + row["samples"]]
            controls = [s for s in controls if s and s.get("known")]
            allocation_failed = row.get('failure_stage') == 'fixture_port_allocation'
            if not controls and not allocation_failed:
                row["passed"], row["policy_error"] = False, "Native stream observations unavailable"
            if not allocation_failed and args.case == "high-latency" and options.get("adaptive_peer_requests"):
                policies = [s.get("request_policy", {}) for s in controls]
                if not any(p.get("enabled") and p.get("qualified_peers", 0) > 0
                           and p.get("horizon_max_ms", 0) > 2000 for p in policies):
                    row["passed"], row["policy_error"] = False, "Measured high-latency policy was not exercised"
            for evidence in controls:
                if (evidence["active_hedges"] > 16 or evidence["peak_hedges"] > 16
                        or evidence["active_hedge_bytes"] > 256 * 1024):
                    row["passed"], row["policy_error"] = False, "Duplicate budget exceeded"
            if row.get("swarm"):
                for peer in row["swarm"].get("peers", []):
                    if peer["peak_pending_requests"] > reqq:
                        row["passed"], row["policy_error"] = False, "Peer capacity exceeded"
            row["stream_counters"] = controls[-1] if controls else None
            row['accepted_trial'] = accepted_trial(args.case, row)
            report["runs"].append(row)
            report["summaries"] = summarize(report["runs"])
            (args.output / "report.json").write_text(
                json.dumps(report, indent=2) + "\n", encoding="utf-8")
            gc.collect()
    report['acceptance'] = ('Every adaptive candidate must complete; stock-horizon '
                            'high-latency control timeouts remain failed observations.')
    (args.output / 'report.json').write_text(json.dumps(report, indent=2)+'\n', encoding='utf-8')
    return all(r['accepted_trial'] for r in report['runs'])


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--executable", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--case", choices=CASES, required=True)
    parser.add_argument("--rotations", type=int, choices=range(2, 7), default=3)
    parser.add_argument("--disk", action="store_true")
    raise SystemExit(0 if check(parser.parse_args()) else 1)
