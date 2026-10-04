#!/usr/bin/env python3
"""Declare Flow's non-virtual cache reconciliation member in the pinned header.

The implementation lives in lt_shim.cpp. No object layout or upstream library
code changes; the static library continues to supply the existing internals.
Use atomic replacement because per-target headers can be hardlinked.
"""
import argparse
import os
from pathlib import Path


def extend(path, anchor, declaration):
    source = path.read_text(encoding="utf-8")
    if declaration in source:
        return
    if source.count(anchor) != 1:
        raise ValueError("Pinned libtorrent cache extension anchor not found exactly once")
    temporary = path.with_suffix(".flow-tmp")
    temporary.write_text(source.replace(anchor, anchor+"\n"+declaration),encoding="utf-8")
    os.replace(temporary,path)


def patch(directory):
    include = directory / "include/libtorrent/aux_"
    if not (include / "torrent.hpp").is_file():
        include = directory / "include/libtorrent"
    extend(include / "torrent.hpp",
           "\t\tvoid set_piece_priority(piece_index_t index, download_priority_t priority);",
           "\t\tvoid flow_forget_piece(piece_index_t index, download_priority_t priority);")
    extend(include / "torrent.hpp",
           "\t\tvoid flow_forget_piece(piece_index_t index, download_priority_t priority);",
           "\t\tvoid flow_refresh_connect_candidates();")
    extend(include / "peer_list.hpp",
           "\t\tvoid set_max_failcount(torrent_state* st);",
           "\t\tvoid flow_refresh_connect_candidates(torrent_state* state) { recalculate_connect_candidates(state); }")
    extend(include / "peer_connection.hpp",
           "\t\tbool has_peer_choked() const { return m_peer_choked; }",
           "\t\tbool flow_snubbed() const { return m_snubbed; }")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("directory",type=Path)
    patch(parser.parse_args().directory)
