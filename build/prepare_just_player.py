#!/usr/bin/env python3
"""Apply the optional Flow patch only to the immutable reviewed player commit."""
import argparse
import hashlib
from pathlib import Path
import subprocess
import shutil

COMMIT = 'aa85148f6ccbfdf931fe207bb75ec77c478d8eb0'
ROOT = Path(__file__).resolve().parent


def prepare(directory):
    directory = directory.resolve(strict=True)
    head = subprocess.check_output(['git', '-C', str(directory), 'rev-parse', 'HEAD'], text=True).strip()
    if head != COMMIT:
        raise ValueError('Just Player checkout must match '+COMMIT)
    patch = ROOT/'just-player/flow-player.patch'
    args = ['git', '-C', str(directory), 'apply', '--whitespace=error']
    if subprocess.run(args+['--reverse', '--check', str(patch)], capture_output=True).returncode:
        subprocess.run(args+['--check', str(patch)], check=True)
        subprocess.run(args+[str(patch)], check=True)
    target = directory/'app/src/main/java/com/brouken/player'
    for source in (ROOT/'just-player').glob('*.java'):
        if source.name.endswith('Test.java'): continue
        (target/source.name).write_bytes(source.read_bytes())
    for source in (ROOT/'just-player/res').glob('*/flow_diagnostics.xml'):
        destination = directory/'app/src/main/res'/source.parent.name/source.name
        destination.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(source,destination)
    # A separate package and label allow stock Just Player to remain installed.
    manifest = directory/'app/src/main/AndroidManifest.xml'
    text = manifest.read_text()
    text = text.replace('android:label="@string/app_name"', 'android:label="Flow Player (experimental)"')
    manifest.write_text(text, encoding='utf-8', newline='\n')
    digest = hashlib.sha256()
    recipe = ROOT/'just-player'
    for source in sorted(path for path in recipe.rglob('*') if path.is_file()):
        digest.update(source.relative_to(recipe).as_posix().encode('utf-8'))
        digest.update(source.read_bytes())
    print('Player commit:',COMMIT,'Flow recipe:',digest.hexdigest())


if __name__ == '__main__':
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('directory', type=Path)
    prepare(parser.parse_args().directory)
