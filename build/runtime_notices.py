"""Preserve original notices for runtimes statically linked by the toolchains."""
import argparse
import os
from pathlib import Path


def collect(target, doc_root=Path('/usr/share/doc'), common_root=Path('/usr/share/common-licenses'), ndk_root=None):
    sections = []

    def add(label, path):
        text = path.read_text(encoding='utf-8')
        if not text.strip():
            raise ValueError(f'Empty runtime notice: {path}')
        sections.append((label, text))

    if target == 'windows-amd64':
        # Ubuntu's installed package copyright files include the MinGW-w64
        # runtime notices and GCC Runtime Library Exception. Preserve their
        # referenced full license texts as well, directly from the build host.
        for package in ('gcc-mingw-w64-base', 'mingw-w64-common'):
            add(f'{package}/copyright', doc_root / package / 'copyright')
        if not (common_root / 'GPL-3').is_file():
            raise ValueError('Missing referenced GCC runtime GPL-3 text')
        for path in sorted(common_root.iterdir()):
            if path.is_file():
                add(f'common-licenses/{path.name}', path)
    elif target in ('android-arm64', 'android-armv7'):
        root = Path(ndk_root or os.environ['ANDROID_NDK_HOME'])
        add('Android NDK/source.properties', root / 'source.properties')
        # NDK distributions ship aggregate notices; LLVM's aggregate includes
        # libc++, libc++abi, libunwind and compiler-rt linked into these builds.
        toolchain = root / 'toolchains/llvm/prebuilt/linux-x86_64'
        found = False
        for path in (root / 'NOTICE', toolchain / 'NOTICE', toolchain / 'NOTICE.txt'):
            if path.is_file():
                add(f'Android NDK/{path.relative_to(root).as_posix()}', path)
                found = True
        if not found:
            raise ValueError('Missing original Android NDK runtime notices')
    else:
        raise ValueError(f'No static toolchain runtime collector for {target}')
    return '\n\n'.join(f'{"=" * 72}\n{name}\n{"=" * 72}\n{text}' for name, text in sections) + '\n'


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--target', required=True)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    notices = collect(args.target)
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(notices, encoding='utf-8')
