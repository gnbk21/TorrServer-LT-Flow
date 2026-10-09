#!/usr/bin/env python3
"""Generate original, legal media fixtures; no downloaded media is needed."""
import argparse
import hashlib
import json
from pathlib import Path
import subprocess


def generate(directory, seconds=12):
    directory.mkdir(parents=True, exist_ok=True)
    variants = (("head.mp4", ["-movflags", "+faststart"]),
                ("tail.mp4", []), ("seekable.mkv", []),
                ("variable.mp4", ["-movflags", "+faststart"]))
    entries = []
    for name, flags in variants:
        target = directory / name
        # Constant quality, changing source complexity: the output is VBR.
        source = "testsrc2=size=640x360:rate=24"
        if name == "variable.mp4":
            source += ",noise=alls=10:allf=t+u"
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
        entries.append({"name": name, "bytes": len(data), "sha256": hashlib.sha256(data).hexdigest(),
                        "duration": float(metadata["format"]["duration"]),
                        "streams": len(metadata["streams"]), "variable_bitrate": True,
                        "mp4_moov_offset": data.find(b"moov") if target.suffix == ".mp4" else None,
                        "mp4_mdat_offset": data.find(b"mdat") if target.suffix == ".mp4" else None})
    head, tail = entries[:2]
    if not (0 <= head["mp4_moov_offset"] < head["mp4_mdat_offset"] and
            tail["mp4_moov_offset"] > tail["mp4_mdat_offset"] >= 0):
        raise RuntimeError("Generated MP4 metadata layout is incorrect")
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
