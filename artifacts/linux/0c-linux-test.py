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
import time
import urllib.error
import urllib.request

BASE = "https://github.com/mcc1/0c-ime/raw/31547683f924412a5f73e359e39fc1f11607b492/artifacts/linux/payload/"
ARCHIVE = "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz"
ARCHIVE_SHA256 = "7c1625841aef280bb21259777fe0999eb535f697d3047a43841d793a6ca80cbb"
PARTS = [
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part00",
        50331648,
        "0bad9a79ffa8a27527980c2865c234b76d5c6e647b4d2e6974f87f0e8a9622a5"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part01",
        50331648,
        "e6c2bca2afb4483acd5b1b4b7105abce253e62110da661c1c78657a05d871b34"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part02",
        50331648,
        "f99a899524aca63d95dacc62110288fe355edd595aff388b5c1e3439952d12f8"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part03",
        50331648,
        "453df6acafc48cebea45cf1d21ea5c770170a323d4016c44988d61124a0f4c4e"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part04",
        50331648,
        "9d0c7f635e565f57ddfa76f40e1d6603278ab2aa5cd43c40b7b8db9dc964e549"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part05",
        50331648,
        "afd6189733447fedc9276b7604888d7c89d05a3ef45a49fa0eba2caba4134a14"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part06",
        50331648,
        "e92d0e8d917b127e8de304ac8213b8f5eee2958ad913b49d72d46806122c56a8"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part07",
        50331648,
        "a8bc834bd85f97a25bf82c6b7af2156c9b5b2aa5649ea5e6013e61a3f303aaa2"
    ],
    [
        "0c-0.2.11-preview.mixed.1-linux-amd64-preview.tar.gz.part08",
        5812417,
        "9ad1cc20980077d6a242d35dc6748877e7c3d145786976872c2cd938cb066c71"
    ]
]


def digest(path):
    h = hashlib.sha256()
    with path.open('rb') as source:
        for block in iter(lambda: source.read(1024 * 1024), b''):
            h.update(block)
    return h.hexdigest()


def obtain_once(directory, name, size, expected):
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


def obtain(directory, name, size, expected):
    for attempt in range(3):
        try:
            return obtain_once(directory, name, size, expected)
        except (urllib.error.URLError, TimeoutError, ConnectionError) as error:
            if isinstance(error, urllib.error.HTTPError) and error.code not in (408, 429, 500, 502, 503, 504):
                raise
            if attempt == 2:
                raise
            print('連線暫時失敗，重試 ' + name + '：' + str(error), flush=True)
            time.sleep(2 ** attempt)


def main():
    parser = argparse.ArgumentParser(description='Kubuntu 26.04 x86_64 混打測試包（含字典、模型與設定介面）')
    parser.add_argument('--install', action='store_true', help='下載校驗並解壓後，執行原作者的使用者目錄安裝器')
    parser.add_argument('--output', default='0c-linux-mixed-input-test', help='下載及解壓目錄')
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
