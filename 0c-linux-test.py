#!/usr/bin/env python3
"""Download and verify the complete Kubuntu mixed-input preview; optionally install it."""
import argparse
import hashlib
import os
from pathlib import Path
import platform
import shutil
import subprocess
import sys
import tempfile
import urllib.request

BASE = "https://github.com/mcc1/0c-ime/raw/715e534c2baf69e641ccc823c439148d39a7d08b/"
ARCHIVE = "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz"
ARCHIVE_SHA256 = "2d74d9567649df3538e6ad43f9ac8c49f30713f04feca7b4db826046cf0b3ee5"
PARTS = [
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part00",
        52428800,
        "06fc3a14e87367bc054698f0e2254c5ef6beb43e1eef2aad337345072832950b"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part01",
        52428800,
        "60a188f0d44c98f353f553b685002d9c045ed91ba2c2f478c0efc474c5e81c41"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part02",
        52428800,
        "9c9e9c4150b30105f7e8c28f51c6c9f89b7fec3faf13eb493834d0fe777c9899"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part03",
        52428800,
        "8fcb024a09df3d87d13dae3ee0c089e4c362076aa66d3d863c34e03ba3ec720d"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part04",
        52428800,
        "d719f8249550ff811e5b30b71bd0589522154f5ca6c62b38eebcfa468e5cc422"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part05",
        52428800,
        "cf374e41409dcf1593dea17267b25d81e1c8e059f1624a680fb1bcb01e0794a9"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part06",
        52428800,
        "746ff846acd8a7dca4d1a14039cec53f448d2f3a340c8b200e43427f1a41f252"
    ],
    [
        "0c-0.2.8-preview.mixedpolicy.1-linux-amd64-preview.tar.gz.part07",
        41864623,
        "d03cbe6ea2650f303a36e944775035647fb052fdf6b1be7063feb80dfc20ca79"
    ]
]


def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as source:
        for block in iter(lambda: source.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def obtain(directory, name, size, expected):
    target = directory / name
    if target.is_file() and target.stat().st_size == size and digest(target) == expected:
        print('已驗證快取：' + name, flush=True)
        return target
    temporary = None
    try:
        with tempfile.NamedTemporaryFile(dir=directory, prefix='.download-', delete=False) as output:
            temporary = Path(output.name)
            count = 0
            print('下載：' + name, flush=True)
            with urllib.request.urlopen(BASE + name, timeout=90) as response:
                while True:
                    block = response.read(1024 * 1024)
                    if not block:
                        break
                    count += len(block)
                    if count > size:
                        raise ValueError('下載超過預期大小：' + name)
                    output.write(block)
        if count != size or digest(temporary) != expected:
            raise ValueError('下載大小或 SHA256 不符：' + name)
        temporary.replace(target)
        return target
    finally:
        if temporary is not None:
            temporary.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description='Kubuntu 26.04 x86_64 混打測試包（含字典、模型與設定介面）')
    parser.add_argument('--install', action='store_true', help='下載校驗並解壓後，執行原作者的使用者目錄安裝器')
    parser.add_argument('--output', default='0c-linux-mixedpolicy-test', help='下載及解壓目錄')
    args = parser.parse_args()
    if sys.platform != 'linux' or platform.machine() != 'x86_64':
        raise ValueError('此套件適用 Linux Intel／AMD 64 位元')
    if args.install and os.getuid() == 0:
        raise ValueError('請以一般使用者執行，不要使用 sudo')
    directory = Path(args.output).resolve()
    directory.mkdir(parents=True, exist_ok=True)
    archive = directory / ARCHIVE
    if not archive.is_file() or digest(archive) != ARCHIVE_SHA256:
        parts = [obtain(directory, *part) for part in PARTS]
        temporary = None
        try:
            with tempfile.NamedTemporaryFile(dir=directory, prefix='.archive-', delete=False) as output:
                temporary = Path(output.name)
                for part in parts:
                    with part.open('rb') as source:
                        shutil.copyfileobj(source, output)
            if digest(temporary) != ARCHIVE_SHA256:
                raise ValueError('完整套件 SHA256 不符')
            temporary.replace(archive)
        finally:
            if temporary is not None:
                temporary.unlink(missing_ok=True)
    print('完整套件 SHA256 驗證通過。', flush=True)
    # Extract only our exact verified archive, into a fresh directory.
    extracted = Path(tempfile.mkdtemp(prefix='unpacked-', dir=directory))
    subprocess.run(['tar', '-xzf', str(archive), '-C', str(extracted)], check=True)
    bundle = extracted / ARCHIVE.removesuffix('.tar.gz')
    if not (bundle / 'install.py').is_file():
        raise ValueError('套件缺少安裝器')
    print('套件目錄：' + str(bundle), flush=True)
    print('請先閱讀：' + str(bundle / 'INSTALL.md'), flush=True)
    if args.install:
        subprocess.run([sys.executable, str(bundle / 'install.py'), 'install'], check=True)
        print('請登出再登入，並在 Fcitx 5 設定加入「靈犀」。', flush=True)
    else:
        print('安裝：python3 ' + repr(str(bundle / 'install.py')) + ' install')


if __name__ == '__main__':
    try:
        main()
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        print('未完成：' + str(error), file=sys.stderr)
        sys.exit(1)
