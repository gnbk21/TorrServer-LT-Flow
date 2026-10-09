#!/usr/bin/env python3
"""Verify preparation against a real pinned checkout, including drift refusal."""
import argparse
from pathlib import Path
import subprocess

from prepare_just_player import prepare


def check(directory):
    prepare(directory)
    source=directory/'app/src/main/java/com/brouken/player/PlayerActivity.java'
    before=source.read_bytes()
    prepare(directory)
    if source.read_bytes()!=before:
        raise AssertionError('Repeated patch changed activity')
    needle=b'player = playerBuilder.build();'
    if before.count(needle)!=1:
        raise AssertionError('Pinned activity marker is ambiguous')
    drift=before.replace(needle,b'player = playerBuilder /* unexpected edit */.build();')
    source.write_bytes(drift)
    try:
        try:
            prepare(directory)
        except subprocess.CalledProcessError:
            pass
        else:
            raise AssertionError('Unexpected activity drift was accepted')
        if source.read_bytes()!=drift:
            raise AssertionError('Rejected patch changed the checkout')
    finally:
        source.write_bytes(before)
    print('Pinned patch repeatability and drift refusal passed')


if __name__=='__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory',type=Path)
    check(parser.parse_args().directory)
