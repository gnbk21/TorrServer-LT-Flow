#!/usr/bin/env python3
"""Apply reviewed native fixes, failing closed on unexpected source context.

Sources may be hardlinked per target. Break links with atomic replacement before
git apply so patches never modify the shared source tree or another target.
The recipe is also part of the installed dependency stamp.
"""
import argparse
import hashlib
import os
from pathlib import Path
import subprocess
import tempfile

PATCH_DIR = Path(__file__).resolve().parent / "native-patches"


def recipe():
    digest = hashlib.sha256()
    for path in [Path(__file__), *sorted(PATCH_DIR.glob("*.patch"))]:
        digest.update(path.name.encode())
        digest.update(path.read_bytes())
    return digest.hexdigest()


def apply(directory):
    directory = directory.resolve(strict=True)
    for patch in sorted(PATCH_DIR.glob("*.patch")):
        args = ["git", "-C", str(directory), "apply", "--whitespace=error"]
        if subprocess.run(args + ["--reverse", "--check", str(patch)],
                          capture_output=True).returncode == 0:
            continue
        checked = subprocess.run(args + ["--check", str(patch)], capture_output=True, text=True)
        if checked.returncode:
            raise RuntimeError(f"Reviewed patch {patch.name} does not match source: {checked.stderr}")
        for line in patch.read_text(encoding="utf-8").splitlines():
            if not line.startswith("+++ b/"):
                continue
            target = (directory / line[6:]).resolve(strict=True)
            if not target.is_relative_to(directory):
                raise ValueError("Patch path escapes source directory")
            with tempfile.NamedTemporaryFile(dir=target.parent, delete=False) as output:
                output.write(target.read_bytes())
                temporary = Path(output.name)
            try:
                os.chmod(temporary, target.stat().st_mode)
                os.replace(temporary, target)
            finally:
                temporary.unlink(missing_ok=True)
        subprocess.run(args + [str(patch)], check=True)
        print(f"Applied {patch.name}")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("directory", type=Path, nargs="?")
    parser.add_argument("--recipe", action="store_true")
    opts = parser.parse_args()
    if opts.recipe:
        print(recipe())
    elif opts.directory:
        apply(opts.directory)
    else:
        parser.error("directory or --recipe required")
