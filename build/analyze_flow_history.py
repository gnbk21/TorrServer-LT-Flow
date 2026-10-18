"""Summarize bounded Flow history without exporting media or client identities."""
import argparse
import collections
import json
import math
import statistics
from pathlib import Path

FILE_LIMIT = 1 << 20


def timing(events, kind):
    values = sorted(e.get("elapsed_ms") for e in events
                    if e.get("type") == kind and e.get("operation_measured")
                    and isinstance(e.get("elapsed_ms"), (int, float))
                    and math.isfinite(e["elapsed_ms"]) and e["elapsed_ms"] >= 0)
    if not values:
        return {"count": 0}
    return {"count": len(values), "median_ms": statistics.median(values),
            "p95_ms": values[max(0, math.ceil(len(values) * .95) - 1)],
            "maximum_ms": values[-1], "over_1000_ms": sum(v > 1000 for v in values)}


def summarize(path):
    events, invalid = [], 0
    # Only the recorder's four exact files, oldest first. No directory scanning.
    for candidate in [Path(str(path) + f".{i}") for i in (3, 2, 1)] + [Path(path)]:
        if not candidate.exists():
            continue
        if candidate.is_symlink() or not candidate.is_file() or candidate.stat().st_size > FILE_LIMIT:
            raise ValueError("history must contain regular files of at most 1 MiB")
        for line in candidate.read_bytes().splitlines():
            try:
                event = json.loads(line)
                if not isinstance(event, dict) or not isinstance(event.get("type"), str) or not isinstance(event.get("run"), str):
                    raise ValueError("not an event")
                events.append(event)
            except (ValueError, UnicodeError):
                invalid += 1
    if not events:
        return {"events": 0, "invalid_lines": invalid, "assessment": "NO_RECORDED_EVIDENCE"}
    run = events[-1].get("run")
    events = [e for e in events if e.get("run") == run]
    counts = collections.Counter(e.get("type") for e in events)
    # Never echo arbitrary event strings, run tokens or nested input fields.
    kinds = ("engine_started", "engine_stopped", "startup", "startup_sample", "playback_sample",
             "first_byte", "recovery", "response", "seek", "peer_disconnected", "peer_summary", "shutdown_requested")
    peers = sum(e.get("count", 0) for e in events if e.get("type") == "peer_summary"
                and isinstance(e.get("count"), int) and e["count"] >= 0)
    snapshots = [e["startup"] for e in events if isinstance(e.get("startup"), dict)]
    blocked = collections.Counter(s.get("blocked_on") for s in snapshots if isinstance(s.get("blocked_on"), str))
    shutdowns = [e for e in events if e.get("type") == "shutdown_requested"]
    reasons = {"API_SHUTDOWN", "SIGNAL", "SERVICE_STOP", "SERVER_ERROR", "SHUTDOWN_TIMEOUT"}
    reason = shutdowns[-1].get("stage") if shutdowns else None
    if not isinstance(reason, str) or reason not in reasons:
        reason = "UNKNOWN"
    closure = "ENGINE_STOP_RECORDED" if counts["engine_stopped"] else "STOP_REQUESTED_WITHOUT_ENGINE_STOP" if shutdowns else "NO_STOP_RECORD"
    samples = {}
    responses = [e["response"] for e in events if e.get("type") == "response" and isinstance(e.get("response"), dict)]
    for event in events:
        sample = event.get("playback")
        if not isinstance(sample, dict):
            continue
        key = (event.get("torrent"), event.get("file"), sample.get("session"))
        if not all(isinstance(value, int) for value in key):
            continue
        samples.pop(key, None)
        samples[key] = sample
    numeric_fields = ("session", "demand_offset_bytes", "delivered_offset_bytes", "buffer_bytes",
                      "buffer_seconds", "delivery_rate", "download_rate", "consumption_rate",
                      "active_readers", "range_requests", "range_cancels", "recent_read_stalls",
                      "recent_wait_p95_ms", "required_piece", "suppliers")
    latest = [{"torrent": key[0], "file": key[1], **{k: v for k, v in s.items() if k in numeric_fields
               and isinstance(v, (int, float)) and math.isfinite(v)}} for key, s in list(samples.items())[-8:]]
    return {"events": len(events), "invalid_lines": invalid,
            "event_counts": {k: counts[k] for k in kinds if counts[k]},
            "peer_disconnects": counts["peer_disconnected"] + peers,
            "response_start": timing(events, "first_byte"), "seek_response_start": timing(events, "recovery"),
            "startup_blocked_samples": {k: blocked[k] for k in ("HEAD", "INDEX", "HEAD_AND_INDEX", "PROBE", "NONE") if blocked[k]},
            "latest_delivery_samples": latest, "closure": closure, "shutdown_reason": reason,
            "responses_cancelled": sum(r.get("cancelled") is True for r in responses),
            "responses_failed": sum(isinstance(r.get("status"), int) and r["status"] >= 400 for r in responses),
            "history_drops": max([0] + [e["dropped"] for e in events if isinstance(e.get("dropped"), int)]),
            "limitations": ["Server response waits are not decoded-player stutters.",
                            "No stop record does not prove a crash; the server may still be running.",
                            "Startup samples are spaced observations, not exact blocked durations.",
                            "Snapshots may omit sessions when truncated; old builds have no periodic snapshots."]}


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("history", type=Path, help="Path to flow-history.jsonl")
    args = parser.parse_args()
    try:
        print(json.dumps(summarize(args.history), indent=2))
    except (OSError, ValueError) as error:
        parser.exit(1, f"Cannot summarize history: {type(error).__name__}\n")
