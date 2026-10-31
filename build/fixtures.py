#!/usr/bin/env python3
"""Generate original, legal media fixtures; no downloaded media is needed."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess
import struct


def mp4_layout(data):
    offsets = {}
    cursor = 0
    while cursor+8 <= len(data):
        size, kind = struct.unpack_from('>I4s', data, cursor)
        header = 8
        if size == 1:
            size = struct.unpack_from('>Q', data, cursor+8)[0]; header = 16
        if size == 0:
            size = len(data)-cursor
        if size < header or cursor+size > len(data):
            raise ValueError('invalid MP4 box')
        if kind in (b'moov', b'mdat'):
            offsets.setdefault(kind.decode(), cursor)
        cursor += size
    return offsets


def mkv_layout(data):
    def vint(at, identifier=False):
        if at >= len(data) or data[at] == 0:
            raise ValueError('invalid EBML integer')
        length = 1
        while not data[at] & (0x80 >> (length-1)):
            length += 1
        if length > 8 or at+length > len(data):
            raise ValueError('truncated EBML integer')
        raw = int.from_bytes(data[at:at+length], 'big')
        value = raw if identifier else raw & ((1 << (7*length))-1)
        return (None if not identifier and value == (1 << (7*length))-1 else value), at+length
    cursor = 0
    while cursor < len(data):
        kind, after = vint(cursor, True); size, payload = vint(after)
        end = len(data) if size is None else payload+size
        if end > len(data):
            raise ValueError('truncated EBML element')
        if kind == 0x18538067:
            cursor = payload
            result = {'cues': None, 'first_cluster': None, 'last_cluster': None}
            while cursor < end:
                kind, after = vint(cursor, True); size, payload = vint(after)
                if size is None or payload+size > end:
                    raise ValueError('fixture needs bounded Matroska elements')
                if kind == 0x1c53bb6b:
                    result['cues'] = cursor
                if kind == 0x1f43b675:
                    if result['first_cluster'] is None:
                        result['first_cluster'] = cursor
                    result['last_cluster'] = cursor
                cursor = payload+size
            return result
        cursor = end
    raise ValueError('Matroska segment missing')


def rate_evidence(target, duration):
    packets = json.loads(subprocess.check_output([
        'ffprobe', '-v', 'error', '-select_streams', 'v:0', '-show_packets',
        '-show_entries', 'packet=pts_time,size', '-of', 'json', str(target)], text=True, timeout=30))
    seconds = {}
    for packet in packets.get('packets', []):
        if packet.get('pts_time') in (None, 'N/A'):
            continue
        second = int(float(packet['pts_time']))
        if 0 <= second < int(duration):
            seconds[second] = seconds.get(second, 0)+int(packet['size'])
    samples = list(seconds.values())
    return {'video_bytes_per_second': samples,
            'variable_bitrate': bool(samples and max(samples)>1.05*min(samples))}


def generate(directory, seconds=12):
    directory.mkdir(parents=True, exist_ok=True)
    variants = (("head.mp4", ["-movflags", "+faststart"]),
                ("tail.mp4", []), ("seekable.mkv", []),
                ("variable.mp4", ["-movflags", "+faststart"]),
                ("front-index.mkv", ["-cues_to_front", "1"]))
    entries = []
    for name, flags in variants:
        target = directory / name
        # Constant quality, changing source complexity: the output is VBR.
        source = "testsrc2=size=640x360:rate=24"
        if name == "variable.mp4":
            source += ",noise=alls=10:allf=t+u:enable='lt(mod(t,4),2)'"
        subprocess.run(["ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
                        "-f", "lavfi", "-i", source, "-f", "lavfi", "-i",
                        "sine=frequency=440:sample_rate=48000", "-t", str(seconds * 5 if name == "variable.mp4" else seconds),
                        "-c:v", "mpeg4", "-q:v", "3", "-c:a", "aac", "-b:a", "64k",
                        "-metadata", "title=Flow generated fixture", *flags, str(target)],
                       check=True, timeout=max(120, seconds * 3))
        metadata = json.loads(subprocess.check_output([
            "ffprobe", "-v", "error", "-show_format", "-show_streams",
            "-print_format", "json", str(target)], text=True, timeout=15))
        data = target.read_bytes()
        mp4 = mp4_layout(data) if target.suffix == '.mp4' else {}
        mkv = mkv_layout(data) if target.suffix == '.mkv' else None
        rates = rate_evidence(target, float(metadata['format']['duration']))
        entries.append({"name": name, "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest(),
                        "duration": float(metadata["format"]["duration"]),
                        "streams": len(metadata["streams"]), **rates,
                        "mp4_moov_offset": mp4.get('moov'),
                        "mp4_mdat_offset": mp4.get('mdat'), 'matroska_layout': mkv})
    head, tail = entries[:2]
    if not (0 <= head["mp4_moov_offset"] < head["mp4_mdat_offset"] and
            tail["mp4_moov_offset"] > tail["mp4_mdat_offset"] >= 0):
        raise RuntimeError("Generated MP4 metadata layout is incorrect")
    tail_mkv, front_mkv = entries[2]['matroska_layout'], entries[4]['matroska_layout']
    if not (tail_mkv['cues'] > tail_mkv['last_cluster'] and front_mkv['cues'] < front_mkv['first_cluster']):
        raise RuntimeError('Generated Matroska index placement is incorrect')
    if not entries[3]['variable_bitrate']:
        raise RuntimeError('Generated variable-rate media did not exercise rate variation')
    (directory / "fixtures.json").write_text(json.dumps({"generator": "ffmpeg lavfi", "copyright": "Original synthetic video and sine tone generated for Flow testing", "fixtures": entries}, indent=2) + "\n", encoding="utf-8")
    return [directory / name for name, _ in variants]


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=Path, required=True)
    parser.add_argument("--seconds", type=int, default=12)
    arguments = parser.parse_args()
    if not 2 <= arguments.seconds <= 3600:
        parser.error("seconds must be between 2 and 3600")
    generate(arguments.output, arguments.seconds)
