#!/usr/bin/env python3
"""Verify historical experiment archives and restore without replacing changed files."""
import argparse
import hashlib
import json
from pathlib import Path
import shutil
import tarfile
import tempfile

def filehash(path):
    h = hashlib.sha256()
    with path.open('rb') as f:
        while block := f.read(1 << 20):
            h.update(block)
    return h.hexdigest()

def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--destination', type=Path, help='original layout root, usually /workspace')
    p.add_argument('--group', action='append', help='archive group; repeat or omit for all')
    p.add_argument('--verify-only', action='store_true')
    a = p.parse_args()
    if not a.verify_only and a.destination is None:
        p.error('--destination or --verify-only is required')
    root = Path(__file__).resolve().parent
    index = json.loads((root / 'archive-index.json').read_text())
    groups = a.group or list(index['records'])
    for name in groups:
        record = index['records'][name]
        with tempfile.TemporaryFile() as joined:
            h = hashlib.sha256()
            for part in record['parts']:
                path = root / part['path']
                if path.stat().st_size != part['size'] or filehash(path) != part['sha256']:
                    raise ValueError('archive part changed: ' + str(path))
                with path.open('rb') as f:
                    while block := f.read(1 << 20):
                        h.update(block)
                        joined.write(block)
            if h.hexdigest() != record['archive_sha256']:
                raise ValueError('joined archive hash changed: ' + name)
            if a.verify_only:
                print(name, 'verified', len(record['files']), 'files')
                continue
            destination = a.destination.resolve()
            expected = {f['path']: f for f in record['files']}
            # Complete preflight before restoring this group. Existing files must
            # be identical; historical sources/locks are never silently rewritten.
            for rel, info in expected.items():
                path = destination / rel
                if Path(rel).is_absolute() or '..' in Path(rel).parts:
                    raise ValueError('unsafe archive path: ' + rel)
                current = destination
                for component in Path(rel).parts:
                    current = current / component
                    if current.is_symlink():
                        raise ValueError('symlink in destination: ' + str(current))
                if path.exists() and (not path.is_file() or filehash(path) != info['sha256']):
                    raise ValueError('existing file differs; use a new destination: ' + str(path))
            joined.seek(0)
            seen = set()
            with tarfile.open(fileobj=joined, mode='r:gz') as tf:
                for member in tf:
                    if not member.isfile() or member.name not in expected or member.name in seen:
                        raise ValueError('unexpected archive member: ' + member.name)
                    seen.add(member.name)
                    info = expected[member.name]
                    path = destination / member.name
                    path.parent.mkdir(parents=True, exist_ok=True)
                    if not path.exists():
                        with path.open('xb') as out, tf.extractfile(member) as src:
                            shutil.copyfileobj(src, out)
                    if path.stat().st_size != info['size'] or filehash(path) != info['sha256']:
                        raise ValueError('restored bytes differ: ' + member.name)
            if seen != set(expected):
                raise ValueError('archive member count differs')
            print(name, 'restored', len(seen), 'files')

if __name__ == '__main__':
    main()
