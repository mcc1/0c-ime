#!/usr/bin/env python3
"""Strict v2 exchange and immutable file helpers; never performs IME inference."""
import gzip
import hashlib
import json
import math
import os
from pathlib import Path
import platform
import subprocess
import unicodedata

import jsonschema

ROOT = Path(__file__).resolve().parents[2]
SCHEMA = json.loads((ROOT / 'design/shift-return-v2/schema.json').read_text())
VALIDATOR = jsonschema.Draft202012Validator(SCHEMA)

class Insufficient(Exception):
    """Necessary data or evidence is unavailable (exit 3)."""

def canonical(value):
    if isinstance(value, float) and value == 0:
        return 0.0
    if isinstance(value, list):
        return [canonical(x) for x in value]
    if isinstance(value, dict):
        return {k: canonical(v) for k, v in value.items()}
    return value

def encoded(value):
    return json.dumps(canonical(value), sort_keys=True, separators=(',', ':'),
                      ensure_ascii=False, allow_nan=False).encode('utf-8')

def digest(value):
    return hashlib.sha256(encoded(value)).hexdigest()

def filehash(path):
    h = hashlib.sha256()
    with open(path, 'rb') as stream:
        for chunk in iter(lambda: stream.read(1 << 20), b''):
            h.update(chunk)
    return h.hexdigest()

def pairs(items):
    result = {}
    for key, value in items:
        if key in result:
            raise ValueError(f'duplicate JSON key {key}')
        result[key] = value
    return result

def parse(data):
    def finite_float(value):
        result=float(value)
        if not math.isfinite(result):raise ValueError('nonfinite JSON number')
        return result
    return json.loads(data, object_pairs_hook=pairs, parse_float=finite_float,
                      parse_constant=lambda x: (_ for _ in ()).throw(ValueError(x)))

def record(path, kind=None):
    value = parse(Path(path).read_bytes())
    VALIDATOR.validate(value)
    if kind is not None and value['kind'] != kind:
        raise ValueError(f'{path}: expected {kind}, got {value["kind"]}')
    if value['kind']=='report':
        for metrics in value['conditions'].values():
            for m in metrics.values():validate_metric(m)
    return value

def validate_metric(m):
    if m['errors']!=m['observed_errors']+m['unavailable_components'] or m['unavailable_components']>m['n'] or m['observed_errors']>m['n']-m['unavailable_components']:raise ValueError('metric charged/observed/unavailable counts inconsistent')
    if m['status']=='complete' and (not m['n'] or m['unavailable_components'] or m['value'] is None or m['wilson95'] is None):raise ValueError('complete metric lacks observed rate')
    if m['status']=='unavailable' and (m['value'] is not None or m['wilson95'] is not None):raise ValueError('unavailable metric fabricates rate/interval')

def records(path, kind):
    opener = gzip.open if str(path).endswith('.gz') else open
    with opener(path, 'rt', encoding='utf-8') as stream:
        for line, data in enumerate(stream, 1):
            if not data.strip():
                raise ValueError(f'{path}:{line}: blank JSONL record')
            value = parse(data)
            VALIDATOR.validate(value)
            if value['kind'] != kind:
                raise ValueError(f'{path}:{line}: expected {kind}')
            yield value

def newout(path):
    path = Path(path)
    path.mkdir(parents=False, exist_ok=False)
    return path

def write(path, value):
    with open(path, 'xb') as stream:
        stream.write(encoded(value) + b'\n')

def verify_lock(path, required=(), check_git=True):
    import numpy
    lock = record(path, 'lock')
    constants = {'max_raw_bytes': 512, 'max_kept_chinese': 256,
                 'max_lex_queries': 250000, 'max_decoder_calls': 100000,
                 'max_pairs': 2000000, 'max_artifact_bytes': 8589934592}
    if any(lock[k] != v for k, v in constants.items()):
        raise ValueError('resource contract differs')
    if lock['baseline_commit'] != 'e754019571573051d8c29e06526802c31d5172e2':
        raise ValueError('baseline commit differs')
    versions = lock['versions']
    actual_go = subprocess.check_output(['go', 'version'], text=True).strip()
    if versions != {'go': actual_go, 'python': platform.python_version(),
                    'numpy': numpy.__version__, 'threads': 1}:
        raise ValueError('tool versions differ from lock')
    for path, expected in lock['files'].items():
        if filehash(path) != expected:
            raise ValueError(f'locked bytes differ: {path}')
    locked = {str(Path(p).resolve()) for p in lock['files']}
    for path in required:
        if str(Path(path).resolve()) not in locked:
            raise ValueError(f'required input missing from lock: {path}')
    if check_git:
        head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
        diff = subprocess.check_output(['git', 'diff', '--binary', 'HEAD'], cwd=ROOT)
        if subprocess.run(['git','merge-base','--is-ancestor',lock['source_commit'],head],cwd=ROOT,stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL).returncode!=0 or hashlib.sha256(diff).hexdigest() != lock['dirty_diff_sha256']:
            raise ValueError('source checkout differs from lock')
    untracked=subprocess.check_output(['git','ls-files','--others','--exclude-standard','-z','core','internal/shiftreturn','cmd/0c-shift-eval','internal/datadir','internal/englishpack','design/shift-return-protocol-v2.md','design/shift-return-source-map-v2.md','design/shift-return-v2/schema.json'],cwd=ROOT)
    if untracked:raise ValueError('implementation source must be committed before a formal lock')
    source_files=subprocess.check_output(['git','ls-files','-z','core','internal/shiftreturn','cmd/0c-shift-eval','internal/datadir','internal/englishpack','design/shift-return-protocol-v2.md','design/shift-return-source-map-v2.md','design/shift-return-v2/schema.json','go.mod','go.sum'],cwd=ROOT).decode().split('\0')
    for source in source_files:
        if source and str((ROOT/source).resolve()) not in locked:raise ValueError(f'source missing from lock: {source}')
    return lock

def normalized_prompt(s):
    return unicodedata.normalize('NFC', s.replace('\r\n', '\n'))
