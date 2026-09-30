#!/usr/bin/env python3
"""Declare Flow's non-virtual cache reconciliation member in the pinned header.

The implementation lives in lt_shim.cpp. No object layout or upstream library
code changes; the static library continues to supply the existing internals.
Use atomic replacement because per-target headers can be hardlinked.
"""
import argparse
import os
from pathlib import Path


def patch(directory):
    path = directory / "include/libtorrent/aux_/torrent.hpp"
    if not path.is_file():
        path = directory / "include/libtorrent/torrent.hpp"
    source = path.read_text(encoding="utf-8")
    declaration = "\t\tvoid flow_forget_piece(piece_index_t index, download_priority_t priority);"
    if declaration in source:
        return
    anchor = "\t\tvoid set_piece_priority(piece_index_t index, download_priority_t priority);"
    if source.count(anchor) != 1:
        raise ValueError("Pinned libtorrent cache extension anchor not found exactly once")
    temporary = path.with_suffix(".flow-tmp")
    temporary.write_text(source.replace(anchor, anchor+"\n"+declaration),encoding="utf-8")
    os.replace(temporary,path)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("directory",type=Path)
    patch(parser.parse_args().directory)
