#!/usr/bin/env python3
"""Strict structural validation and transitive split construction (mixed-family-v1).

Requires the host's jsonschema package; no network or personal files are read.
"""
import hashlib
import json
import pathlib
import sys
import unicodedata
from collections import Counter
import jsonschema


def pairs(items):
    d = {}
    for k, v in items:
        if k in d:
            raise ValueError(f'duplicate JSON key: {k}')
        d[k] = v
    return d


def read(path):
    return json.loads(pathlib.Path(path).read_text('utf-8'), object_pairs_hook=pairs,
                      parse_constant=lambda x: (_ for _ in ()).throw(ValueError(x)))


def norm(s):
    s = unicodedata.normalize('NFC', s).strip(' \t\r\n\v\f')
    return s.translate(str.maketrans('ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'))


def digest(b):
    return hashlib.sha256(b).hexdigest()


def construct(fixtures, aliases, schema):
    raw = pathlib.Path(fixtures).read_bytes()
    if not raw.endswith(b'\n') or b'\r' in raw:
        raise ValueError('fixtures must be UTF-8 LF with final LF')
    validator = jsonschema.Draft202012Validator(read(schema))
    rows = []
    ids = set()
    for line in raw.decode('utf-8').splitlines():
        r = json.loads(line, object_pairs_hook=pairs, parse_constant=lambda x: (_ for _ in ()).throw(ValueError(x)))
        validator.validate(r)
        if r['id'] in ids:
            raise ValueError('duplicate fixture id')
        ids.add(r['id'])
        rows.append(r)
    a = read(aliases)
    if set(a) != {'version', 'aliases'} or a['version'] != 'mixed-family-v1' or not isinstance(a['aliases'], dict):
        raise ValueError('aliases structure')
    aa = {}
    for k, v in a['aliases'].items():
        k, v = norm(k), norm(v)
        if k in aa or k == v or k.split(':')[0] != v.split(':')[0] or k.split(':')[0] not in {'prefix','word','literal','template','source','context'}:
            raise ValueError('invalid alias')
        aa[k] = v
    if set(aa) & set(aa.values()):
        raise ValueError('alias chain/cycle')
    parent = list(range(len(rows)))
    def find(i):
        while parent[i] != i:
            parent[i] = parent[parent[i]]
            i = parent[i]
        return i
    def union(i,j):
        parent[find(j)] = find(i)
    seen = {}
    edges = []
    conditions = {}
    for i,r in enumerate(rows):
        rawkeys = ''.join(e['key'] for e in r['events'] if e['type'] == 'key')
        keys = {aa.get(norm(f), norm(f)) for f in r['families']}
        keys.add('auto:raw:' + norm(rawkeys))
        for token in r['custom_tokens']:
            keys.add('auto:custom:' + norm(token))
        if r['prelude']:
            keys.add('auto:context:' + json.dumps(r['prelude'], ensure_ascii=False, sort_keys=True, separators=(',',':')))
        partitions = []
        for assertion in r['assertions']:
            if assertion['stage'] != 'final' or assertion['policy'] != 'generator':
                continue
            for p in assertion['accepted']:
                partitions.append(p)
                chinese = [s for s in p if s['kind'] == 'chinese']
                if chinese:
                    prefix = ''.join(s['text'] for s in p if s['kind']=='literal' and s['end'] <= chinese[0]['begin'])
                    keys.add('auto:prefix-reading:' + json.dumps([norm(prefix), [s['reading'] for s in chinese]],ensure_ascii=False,separators=(',',':')))
                elif r['intent'] == 'literal':
                    keys.add('auto:literal:' + norm(''.join(s['text'] for s in p)))
        cond = json.dumps([r['prelude'],r['custom_tokens'],r['events']],ensure_ascii=False,sort_keys=True,separators=(',',':'))
        gold = json.dumps(partitions,ensure_ascii=False,sort_keys=True,separators=(',',':'))
        if r['intent'] != 'ambiguous' and cond in conditions and conditions[cond] != (r['intent'],gold):
            raise ValueError('contradictory complete-condition labels: '+r['id'])
        if r['intent'] != 'ambiguous':
            conditions[cond] = (r['intent'],gold)
        for k in keys:
            if k in seen:
                union(i,seen[k])
            seen[k] = i
        edges.append(keys)
    components = {}
    for i in range(len(rows)):
        components.setdefault(find(i),[]).append(i)
    assigned = {}
    for members in components.values():
        keys = sorted(set().union(*(edges[i] for i in members)),key=lambda x:x.encode('utf-8'))
        identity = '\n'.join(json.dumps(k,ensure_ascii=False,separators=(',',':')) for k in keys)
        gid = 'g:' + digest(identity.encode('utf-8'))
        bucket = int.from_bytes(hashlib.sha256(('mixed-family-v1\nseed=20261101\n'+gid).encode()).digest()[:8],'big') % 10
        split = 'train' if bucket < 6 else 'dev' if bucket < 8 else 'test'
        for i in members:
            assigned[rows[i]['id']] = {'group_id':gid, 'split':split}
    assignment = {'version':'mixed-family-v1','seed':'20261101','fixtures_sha256':digest(raw),
                  'aliases_sha256':digest(pathlib.Path(aliases).read_bytes()),'rows':assigned}
    counts = {}
    primary = {}
    for split in ('train','dev','test'):
        counts[split] = {}
        for intent in ('chinese','literal','ambiguous'):
            subset = [r for r in rows if r['intent']==intent and assigned[r['id']]['split']==split]
            groups = {}
            for r in subset:
                groups.setdefault(assigned[r['id']]['group_id'],[]).append(r['id'])
            counts[split][intent] = {'rows':len(subset),'groups':len(groups)}
            if intent != 'ambiguous':
                for gid, ri in groups.items():
                    primary[split+':'+intent+':'+gid] = min(ri,key=lambda rid:(digest(('primary-v1\n'+rid).encode()),rid))
    pilot = all(counts[s][k]['groups'] >= (3 if k=='ambiguous' else 10) for s in counts for k in counts[s]) and all(sum(counts[s][k]['rows'] for s in counts)>=n for k,n in [('chinese',100),('literal',100),('ambiguous',30)])
    release = counts['test']['chinese']['groups'] >= 400 and counts['test']['literal']['groups'] >= 400
    return assignment, {'counts':counts,'components':len(components),'pilot_adequate':pilot,'release_adequate':release,'primary':primary}


def main():
    mode,fixtures,aliases,schema,assignment = sys.argv[1:]
    expected, stats = construct(fixtures,aliases,schema)
    target = pathlib.Path(assignment)
    if mode=='assign':
        with target.open('x',encoding='utf-8',newline='\n') as f:
            f.write(json.dumps(expected,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    elif mode=='validate':
        if read(target) != expected:
            raise ValueError('assignment does not match immutable family construction')
        if target.read_text('utf-8') != json.dumps(expected,ensure_ascii=False,sort_keys=True,indent=2)+'\n':
            raise ValueError('assignment is not canonical UTF-8 sorted indent2 LF')
    else:
        raise ValueError('unknown phase')
    print(json.dumps(stats,ensure_ascii=False,sort_keys=True))

if __name__ == '__main__':
    main()
