#!/usr/bin/env python3
import sys
import json,pathlib
B=pathlib.Path(sys.argv[2]).resolve();R=pathlib.Path(sys.argv[1]).resolve()
rows=[json.loads(s) for s in (B/'test-v1/rows.jsonl').read_text().splitlines()]
fixtures={r['id']:r for r in map(json.loads,(R/'testdata/mixed-boundary/fixtures.jsonl').read_text().splitlines())}
chosen=['mandatory-chinese-00','mandatory-chinese-02','mandatory-context','pciids-1402-6144-00']
for cat in ['direct','unknown-prefix','invalid-first-resumption']:
    options=[r for r in rows if r['primary'] and r['intent']=='chinese' and cat in r['categories'] and r['first_boundary_error']['proposed'] and not r['first_boundary_error']['aggressive']]
    if options:chosen.append(sorted(options,key=lambda r:r['fixture_id'])[0]['fixture_id'])
# Read exact event strings; never infer raw ownership from displayed text.
out={}
for rid in chosen:
    f=fixtures[rid];out[rid]={'raw':''.join(e['key'] for e in f['events'] if e['type']=='key'),'prelude_raw':''.join(e['key'] for e in f['prelude'] if e['type']=='key'),'word_family':[x for x in f['families'] if x.startswith('word:')],'aggressive':[],'proposed':[]}
for line in (B/'enumerate-v1/events.jsonl').open():
    e=json.loads(line);rid=e['fixture_id']
    if rid not in out or e['policy']!='aggressive':continue
    out[rid]['aggressive'].append({'event':e['event'],'visible':e['visible'],'commit':e['commit'],'partition':e['partition'],'ownership_status':e['ownership_status']})
for line in (B/'test-v1/events.jsonl').open():
    e=json.loads(line);rid=e['fixture_id']
    if rid not in out:continue
    tr=e['transition'];out[rid]['proposed'].append({'event':e['event'],'visible':e['visible'],'commit':tr['commit'],'partition':tr['after']['partition'] if tr['after'] else None,'selected_lane':e['selected_lane'],'revision_count':tr['after']['revision_count'] if tr['after'] else None})
(B/'examples-v1.json').write_text(json.dumps(out,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
for rid,r in out.items():
    a=[e for e in r['aggressive'] if e['commit']];p=[e for e in r['proposed'] if e['commit'] is not None]
    print(rid,repr(r['prelude_raw']),repr(r['raw']),'word',r['word_family'],'A=',repr(a[-1]['commit'] if a else None),'E_run=',repr(p[-1]['commit'] if p else None))
