#!/usr/bin/env python3
"""Independent read-only public dictionary parser for qualification, never decoding."""
import hashlib
import math
from pathlib import Path
import struct
import zlib
from exchange import parse

def public_dictionary(directory, locked=None):
    directory = Path(directory)
    def checked(path, expected):
        data = path.read_bytes()
        if hashlib.sha256(data).hexdigest() != expected:
            raise ValueError(f'dictionary manifest hash mismatch {path}')
        if locked is not None and str(path.resolve()) not in locked:
            raise ValueError(f'dictionary absent from frozen lock {path}')
        return data
    manifest = parse((directory/'manifest.json').read_bytes())
    data = checked(directory/'lexicon.bin', manifest['files']['lexicon.bin']['sha256'])
    # This execution source is the verified plaintext public pack. Sealed packs
    # require a separately locked source adapter, never silent fallback data.
    if data[:8] != b'ZUINLEX\0' or struct.unpack_from('<I', data, 8)[0] != 1:
        raise ValueError('unsupported public lexicon representation')
    if zlib.crc32(data[:-4]) != struct.unpack_from('<I', data, len(data)-4)[0]:
        raise ValueError('lexicon CRC mismatch')
    offset = 12
    def section(fmt):
        nonlocal offset
        count = struct.unpack_from('<I', data, offset)[0]; offset += 4
        size = struct.calcsize(fmt)*count
        if offset+size > len(data)-4:
            raise ValueError('truncated lexicon section')
        result = struct.unpack_from('<'+str(count)+fmt, data, offset)
        offset += size
        return result
    syllables = bytes(section('B')).decode().split()
    readings = section('H')
    reading_offsets, surface_offsets = section('I'), section('I')
    surface_lengths, reading_lengths, flags, weights = section('H'), section('B'), section('B'), section('f')
    surfaces = bytes(section('B'))
    by_text = section('I')
    if offset != len(data)-4 or len({len(x) for x in (reading_offsets, surface_offsets, surface_lengths, reading_lengths, flags, weights, by_text)}) != 1:
        raise ValueError('lexicon dimensions/trailing bytes differ')
    entries = []
    for ro, so, sl, rl, weight in zip(reading_offsets, surface_offsets, surface_lengths, reading_lengths, weights):
        if rl == 0 or ro+rl > len(readings) or so+sl > len(surfaces) or not 0 < weight <= 1:
            raise ValueError('invalid lexicon entry')
        reading = tuple(syllables[i] for i in readings[ro:ro+rl])
        entries.append((surfaces[so:so+sl].decode(), reading))
    english_manifest = parse((directory/'english/manifest.json').read_bytes())
    english = checked(directory/'english/words.bin', english_manifest['files']['words.bin']['sha256'])
    if english[:6] != b'0cEN\0\x01':
        raise ValueError('unsupported public English representation')
    count = struct.unpack_from('<I', english, 6)[0]; at = 10; words = set(); previous = ''
    for _ in range(count):
        length = english[at]; at += 1
        text = english[at:at+length].decode('ascii'); at += length
        weight = struct.unpack_from('<d', english, at)[0]; flag = english[at+8]; at += 9
        if not text.isalpha() or not text.islower() or text <= previous or not math.isfinite(weight) or weight <= 0 or flag > 1:
            raise ValueError('invalid English entry')
        words.add(text); previous = text
    if at != len(english):
        raise ValueError('trailing English bytes')
    return entries, words
