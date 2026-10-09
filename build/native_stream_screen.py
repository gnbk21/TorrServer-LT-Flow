"""Local native scheduling screen; NOT Flow cache, HTTP, or decoded playback."""
import argparse
import hashlib
import json
from pathlib import Path
import sys
import tempfile
import time

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--binding-path", type=Path, help="Optional directory containing the libtorrent package")
parser.add_argument("--output", type=Path, required=True, help="JSON report; scratch files use its directory")
parser.add_argument("--rotations", type=int, choices=range(2, 13), default=2)
args = parser.parse_args()
if args.binding_path:
    sys.path.insert(0, str(args.binding_path.resolve()))
import libtorrent as lt
from controlled_peer import LocalSwarm, PeerPlan

if lt.__version__ != "2.1.2.0":
    raise SystemExit("This screen targets the pinned native 2.1.2.0 API")
args.output.parent.mkdir(parents=True, exist_ok=True)

PIECE = 4 * 1024 * 1024
RATE = 11_245_516
DURATION = 8
WINDOW = 32


def run(source, policy, seed):
    # Several slow suppliers plus faster suppliers; a fast supplier disappears.
    plans = [PeerPlan(rate=128 * 1024) for _ in range(8)]
    plans += [PeerPlan(rate=1536 * 1024) for _ in range(8)]
    plans[8 + seed % 8].outages = [(4, 7)]
    with tempfile.TemporaryDirectory(dir=args.output.parent, prefix="native-screen-") as folder:
        with LocalSwarm([source], peers=plans, piece_length=PIECE, peer_ip_start=40) as swarm:
            session = lt.session({
                "enable_dht": False, "enable_lsd": False, "enable_upnp": False,
                "enable_natpmp": False, "listen_interfaces": "127.0.0.1:0",
                "outgoing_interfaces": "127.0.0.1", "enable_outgoing_utp": False,
                "enable_incoming_tcp": False, "enable_incoming_utp": False,
                "allow_multiple_connections_per_ip": True,
                "close_redundant_connections": False, "connections_limit": 50,
                "request_queue_time": 1, "max_out_request_queue": 1500,
                "max_allowed_in_request_queue": 2000, "connection_speed": 250,
                "torrent_connect_boost": 100, "peer_connect_timeout": 7,
                "piece_timeout": 10, "min_reconnect_time": 10,
                "strict_end_game_mode": True, "prioritize_partial_pieces": False,
            })
            params = lt.add_torrent_params()
            params.ti = lt.torrent_info(lt.bencode({b"info": swarm.info}))
            params.save_path = folder
            params.flags = lt.torrent_flags.paused | lt.torrent_flags.sequential_download
            handle = session.add_torrent(params)
            old_deadlines = set()
            last_prios = []

            def schedule(head):
                nonlocal old_deadlines, last_prios
                priorities = [0] * swarm.piece_count
                for piece in range(head, min(swarm.piece_count, head + WINDOW)):
                    distance = piece - head
                    priorities[piece] = 7 if distance == 0 else 6 if distance == 1 else 5 if distance <= 3 else 4 if distance <= 8 else 3
                lead = 8 if policy == "bounded-8" else WINDOW
                desired = set(range(head, min(swarm.piece_count, head + lead)))
                if policy == "graded-after":
                    for piece in old_deadlines - desired:
                        handle.reset_piece_deadline(piece)
                if priorities != last_prios:
                    handle.prioritize_pieces(priorities)
                    last_prios = priorities
                if policy != "graded-after":
                    for piece in old_deadlines - desired:
                        handle.reset_piece_deadline(piece)
                for piece in sorted(desired):
                    distance = piece - head
                    deadline = 0 if distance == 0 else 100 if distance == 1 else 500 if distance <= 3 else 1500 if distance <= 8 else 1500 + (distance - 8) * 500
                    handle.set_piece_deadline(piece, deadline)
                if policy == "graded-after":
                    handle.prioritize_pieces(priorities)
                old_deadlines = desired

            schedule(0)
            handle.resume()
            for peer in swarm.peers:
                handle.connect_peer(peer.server_address)
            began = time.monotonic()
            bootstrap = None
            offset = 0
            waits = []
            samples = []
            next_schedule = began + 1
            playback_began = None
            next_sample = began
            while time.monotonic() - began < 24:
                now = time.monotonic()
                head = offset // PIECE
                if now >= next_schedule:
                    schedule(head)
                    next_schedule = now + 1
                if now >= next_sample:
                    state = handle.status()
                    pieces = state.pieces
                    complete = head
                    while complete < len(pieces) and pieces[complete]:
                        complete += 1
                    samples.append({"seconds": round(now-began, 3), "offset": offset,
                                    "contiguous_bytes": max(0, complete*PIECE-offset),
                                    "payload_bytes": state.total_payload_download,
                                    "peers": state.num_peers})
                    next_sample = now + 1
                if playback_began is None:
                    if handle.have_piece(0) and handle.have_piece(1):
                        playback_began = now
                        bootstrap = now - began
                    else:
                        time.sleep(.01)
                        continue
                if now - playback_began >= DURATION:
                    break
                due = int((now-playback_began) * RATE)
                wanted = min(due, DURATION * RATE)
                if wanted > offset:
                    needed = (wanted - 1) // PIECE
                    if all(handle.have_piece(piece) for piece in range(offset//PIECE, needed+1)):
                        offset = wanted
                    else:
                        waits.append(now)
                time.sleep(.01)
            state = handle.status()
            session.pause()
            # Check a retained prefix byte for byte. The disk writer is complete
            # for have_piece pieces, which are the only pieces counted as read.
            output = Path(folder) / "Flow-generated-fixtures" / source.name
            assert bootstrap is not None, "native bootstrap timed out"
            assert offset > 0, "no paced bytes became available"
            with output.open("rb") as stream:
                checked = stream.read(offset)
            assert checked == swarm.data[:offset]
            result = {"policy": policy, "rotation": seed, "startup_seconds": bootstrap,
                      "read_bytes": offset, "verified_prefix": True,
                      "blocked_poll_count": len(waits),
                      "elapsed_seconds": round(time.monotonic()-began, 3),
                      "payload_bytes": state.total_payload_download,
                      "peer_sent_bytes": swarm.status()["sent_bytes"], "samples": samples}
            session.remove_torrent(handle)
            del handle
            del session
            return result


report = {"native_version": lt.__version__, "upstream_binding_only": True,
          "flow_cache_http_test": False, "decoded_playback_test": False,
          "demand_bytes_per_second": RATE, "piece_bytes": PIECE,
          "playback_seconds": DURATION, "window_pieces": WINDOW, "cases": []}
with tempfile.TemporaryDirectory(dir=args.output.parent, prefix="native-source-") as folder:
    source = Path(folder) / "synthetic-bytes.bin"
    # Deterministic original byte fixture, not user media or a downloaded torrent.
    with source.open("wb") as stream:
        for index in range(48):
            stream.write(hashlib.sha256(str(index).encode()).digest() * (PIECE // 32))
    report["source_bytes"] = source.stat().st_size
    report["source_sha256"] = hashlib.sha256(source.read_bytes()).hexdigest()
    for rotation in range(args.rotations):
        policies = ["full", "bounded-8", "graded-after"]
        shift = rotation % len(policies)
        policies = policies[shift:] + policies[:shift]
        for policy in policies:
            result = run(source, policy, rotation)
            report["cases"].append(result)
            args.output.write_text(json.dumps(report, indent=2)+"\n", encoding="utf-8")
            print(json.dumps({key: value for key, value in result.items() if key != "samples"}), flush=True)
