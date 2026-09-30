#!/usr/bin/env python3
"""Classify Flow tags and package the exact CI binaries, using only the stdlib."""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import zipfile

REPOSITORY = "gnbk21/TorrServer-LT-Flow"
PLATFORMS = ("windows-amd64", "linux-amd64", "linux-arm64", "linux-armv7",
             "android-arm64", "android-armv7", "darwin-amd64", "darwin-arm64")
GST = {"windows-amd64", "linux-amd64", "linux-arm64", "darwin-amd64", "darwin-arm64"}


def classify(tag):
    if re.fullmatch(r"MatriX\.145\.Flow-v\d+\.\d+\.\d+", tag):
        return "stable"
    if re.fullmatch(r"MatriX\.145\.Flow-v\d+\.\d+\.\d+-(preview|alpha|beta|rc)\.\d+", tag) or re.fullmatch(r"MatriX\.145\.Flow-preview\.\d+", tag):
        return "preview"
    raise ValueError("Use MatriX.145.Flow-vX.Y.Z or MatriX.145.Flow-vX.Y.Z-preview.N")


def digest(path):
    with path.open("rb") as source:
        return hashlib.file_digest(source, "sha256").hexdigest()


def module_notices(binaries, native_root):
    modules = set()
    for binary in binaries:
        info = subprocess.check_output(["go", "version", "-m", str(binary)], text=True)
        modules.update(re.findall(r"^\s*dep\s+(\S+)\s+(\S+)", info, re.M))
    cache = Path(subprocess.check_output(["go", "env", "GOMODCACHE"], text=True).strip())
    goroot = Path(subprocess.check_output(["go", "env", "GOROOT"], text=True).strip())
    sections = [("Go runtime", (goroot / "LICENSE").read_text(encoding="utf-8"))]
    for name, version in sorted(modules):
        escaped = re.sub(r"[A-Z]", lambda m: "!" + m[0].lower(), name)
        directory = cache / f"{escaped}@{version}"
        notices = [p for p in directory.iterdir() if p.is_file() and re.match(r"^(LICENSE|LICENCE|COPYING|NOTICE|COPYRIGHT)([._-].*)?$", p.name, re.I)]
        if not notices:
            raise ValueError(f"Missing original license notice: {name}@{version}")
        sections.extend((f"{name}@{version}/{p.name}", p.read_text(encoding="utf-8", errors="replace")) for p in sorted(notices))
    # License files from pinned native sources, including nested WebRTC deps.
    # No network-dependent license lookup is needed during packaging.
    native_notices = []
    for base in native_root.iterdir():
        if not base.is_dir():
            continue
        for directory, dirs, files in os.walk(base):
            dirs[:] = [d for d in dirs if d not in {".git", "doc", "docs", "test", "tests", "example", "examples"}]
            for name in files:
                if re.match(r"^(LICENSE|LICENCE|COPYING|NOTICE|COPYRIGHT)([._-].*)?$", name, re.I) and not name.endswith((".html", ".rst")):
                    path = Path(directory) / name
                    native_notices.append((path.relative_to(native_root).as_posix(), path.read_text(encoding="utf-8", errors="replace")))
    for component in ("boost_", "openssl-", "libtorrent/"):
        if not any(name.startswith(component) for name, _ in native_notices):
            raise ValueError(f"Missing native license sources: {component}")
    sections.extend(sorted(native_notices))
    return "\n\n".join(f"{'=' * 72}\n{name}\n{'=' * 72}\n{text}" for name, text in sections)


def package(artifact_dir, output, tag, commit, native_root):
    channel = classify(tag)
    if not re.fullmatch(r"[a-f0-9]{40}", commit):
        raise ValueError("Expected full build commit")
    output.mkdir(parents=True, exist_ok=False)
    root = Path(__file__).resolve().parent.parent
    binaries = []
    files = {}
    for platform in PLATFORMS:
        names = [f"TorrServer-LT-{platform}" + (".exe" if platform.startswith("windows") else "")]
        if platform in GST:
            names.append(f"TorrServer-LT-{platform}-gst" + (".exe" if platform.startswith("windows") else ""))
        for name in names:
            found = list(artifact_dir.rglob(name))
            if len(found) != 1:
                raise ValueError(f"Expected exactly one {name}")
            binary = found[0]
            info = subprocess.check_output(["go", "version", "-m", str(binary)], text=True)
            if f"vcs.revision={commit}" not in info or f"server/version.Version={tag}" not in info:
                raise ValueError(f"Binary identity mismatch: {name}")
            binaries.append(binary)
            files[name] = {"sha256": digest(binary), "size": binary.stat().st_size}
            (output / name).write_bytes(binary.read_bytes())
    base_url = f"https://github.com/{REPOSITORY}/releases/download/{tag}/"
    provenance = {"schema_version": 1, "repository": REPOSITORY, "source_commit": commit,
                  "version": tag, "channel": channel, "binaries": files,
                  "source_url": f"https://github.com/{REPOSITORY}/tree/{commit}"}
    provenance["native_build_pins"] = (root / "build/_common.sh").read_text(encoding="utf-8")
    provenance["native_cache_extension"] = {"version": 2, "declarations_sha256": digest(root / "build/patch_libtorrent_header.py"), "implementation_sha256": digest(root / "server/lt/lt_shim.cpp")}
    (output / "BUILDINFO.json").write_text(json.dumps(provenance, indent=2) + "\n", encoding="utf-8")
    (output / "NATIVE_AND_GO_NOTICES.txt").write_text(module_notices(binaries, native_root), encoding="utf-8")
    for name in ("LICENSE", "DISTRIBUTION.md"):
        (output / name).write_bytes((root / name).read_bytes())
    (output / "THIRD_PARTY_NOTICES.txt").write_bytes((root / "server/web/pages/template/pages/THIRD_PARTY_NOTICES.txt").read_bytes())
    (output / "FlowTray.ps1").write_bytes((root / "tray/FlowTray.ps1").read_bytes())
    for name in ("Install-Flow.ps1", "Update-Flow.ps1", "FlowRelease.ps1"):
        (output / name).write_bytes((root / "distribution" / name).read_bytes())
    documents = ("LICENSE", "DISTRIBUTION.md", "BUILDINFO.json", "NATIVE_AND_GO_NOTICES.txt", "THIRD_PARTY_NOTICES.txt")
    packages = {}
    for platform in PLATFORMS:
        name = f"TorrServer-Flow-{platform}-{tag}.zip"
        with zipfile.ZipFile(output / name, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
            def add(path, executable=False):
                entry = zipfile.ZipInfo(path.name, (1980, 1, 1, 0, 0, 0))
                entry.compress_type = zipfile.ZIP_DEFLATED
                entry.create_system = 3
                entry.external_attr = ((0o100755 if executable else 0o100644) << 16)
                archive.writestr(entry, path.read_bytes(), compresslevel=9)
            contents = [(output / doc, False) for doc in documents]
            for binary in binaries:
                if binary.name.startswith(f"TorrServer-LT-{platform}"):
                    contents.append((binary, True))
            if platform.startswith("windows"):
                for script in ("FlowTray.ps1", "Install-Flow.ps1", "Update-Flow.ps1", "FlowRelease.ps1"):
                    contents.append((output / script, False))
            for path, executable in sorted(contents, key=lambda item: item[0].name):
                add(path, executable)
        packages[platform] = {"name": name, "url": base_url + name, "sha256": digest(output / name), "size": (output / name).stat().st_size}
    build_epoch = int(subprocess.check_output(["git", "show", "-s", "--format=%ct", commit], cwd=root, text=True).strip())
    manifest = {"Name": "TorrServer-Flow", "Version": tag, "BuildDate": datetime.datetime.fromtimestamp(build_epoch, datetime.timezone.utc).isoformat(),
                "schema_version": 1, "repository": REPOSITORY, "channel": channel, "commit": commit,
                "Links": {p + (".exe" if p.startswith("windows") else ""): base_url + "TorrServer-LT-" + p + (".exe" if p.startswith("windows") else "") for p in PLATFORMS},
                "files": {n: dict(v, url=base_url+n) for n,v in files.items()}, "packages": packages}
    (output / "release.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")
    (output / "SHA256SUMS").write_text("".join(f"{digest(p)}  {p.name}\n" for p in sorted(output.iterdir()) if p.is_file()), encoding="utf-8")
    print(json.dumps({"channel": channel, "binaries": len(binaries), "packages": len(packages)}))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--tag", required=True)
    parser.add_argument("--classify", action="store_true")
    parser.add_argument("--artifacts", type=Path)
    parser.add_argument("--output", type=Path)
    parser.add_argument("--commit")
    parser.add_argument("--native-sources", type=Path)
    args = parser.parse_args()
    if args.classify:
        print(classify(args.tag))
    else:
        if not all((args.artifacts, args.output, args.commit, args.native_sources)):
            parser.error("packaging requires --artifacts --output --commit --native-sources")
        package(args.artifacts, args.output, args.tag, args.commit, args.native_sources)
