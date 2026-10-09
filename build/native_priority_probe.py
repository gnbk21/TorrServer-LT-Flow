"""Probe upstream native API semantics, not Flow throughput or decoded playback."""
import argparse
import hashlib
import json
from pathlib import Path
import sys
import tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--binding-path", type=Path, help="Optional directory containing the libtorrent package")
parser.add_argument("--output", type=Path, required=True, help="JSON report; scratch files use its directory")
args = parser.parse_args()
if args.binding_path:
    sys.path.insert(0, str(args.binding_path.resolve()))
import libtorrent as lt

if lt.__version__ != "2.1.2.0":
    raise SystemExit("This probe targets the pinned native 2.1.2.0 API")
args.output.parent.mkdir(parents=True, exist_ok=True)

block = bytes(range(256)) * 64
metadata = {b"info": {b"name": b"flow-priority-probe.bin", b"piece length": len(block),
                       b"length": len(block) * 32,
                       b"pieces": hashlib.sha1(block).digest() * 32}}
base = [7, 6, 5, 5, 4, 4, 4, 4, 4] + [3] * 23
report = {"native_version": lt.__version__, "upstream_binding_only": True,
          "flow_throughput_test": False, "decoded_playback_test": False, "cases": []}
with tempfile.TemporaryDirectory(dir=args.output.parent, prefix="native-priority-") as scratch:
    session = lt.session({"enable_dht": False, "enable_lsd": False,
                          "enable_upnp": False, "enable_natpmp": False,
                          "listen_interfaces": "127.0.0.1:0",
                          "enable_outgoing_tcp": False, "enable_outgoing_utp": False,
                          "enable_incoming_tcp": False, "enable_incoming_utp": False})
    for lead in (32, 8):
        params = lt.add_torrent_params()
        params.ti = lt.torrent_info(lt.bencode(metadata))
        params.save_path = scratch
        params.flags = lt.torrent_flags.paused
        handle = session.add_torrent(params)
        handle.prioritize_pieces(base)
        before = list(handle.get_piece_priorities())
        for piece in range(lead):
            handle.set_piece_deadline(piece, piece * 500)
        after = list(handle.get_piece_priorities())
        assert before == base
        assert after[:lead] == [7] * lead
        assert after[lead:] == base[lead:]
        handle.clear_piece_deadlines()
        after_clear = list(handle.get_piece_priorities())
        assert after_clear[:lead] == [1] * lead
        assert after_clear[lead:] == base[lead:]
        handle.prioritize_pieces(base)
        restored = list(handle.get_piece_priorities())
        assert restored == base
        report["cases"].append({"deadline_pieces": lead, "before": before,
                                 "after_deadlines": after, "after_clear": after_clear,
                                 "after_explicit_restore": restored})
        session.remove_torrent(handle)
    session.pause()
    del handle
    del session
report["passed"] = True
args.output.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
print(json.dumps({"native_version": lt.__version__, "cases": len(report["cases"]),
                  "passed": True, "output": str(args.output)}))
