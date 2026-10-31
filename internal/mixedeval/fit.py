#!/usr/bin/env python3
"""The single preregistered mixed-fit-v1 estimator. No held-out scores here."""
import os
os.environ['OPENBLAS_NUM_THREADS']='1'
os.environ['OMP_NUM_THREADS']='1'
import hashlib
import json
import pathlib
import sys
import numpy as np


def sha(path):return hashlib.sha256(pathlib.Path(path).read_bytes()).hexdigest()

def identity(p):return json.dumps([[s['kind'],s['begin'],s['end'],s.get('reading','')] for s in p],ensure_ascii=False,separators=(',',':'))

def project(p,end):
    out=[]
    for s in p:
        if s['begin']>=end:break
        z=dict(s);z['end']=min(z['end'],end);out.append(z)
    return identity(out)

def train(root,phase1,out):
    config_path=root/'testdata/mixed-boundary/fitting-v1.json';config=json.loads(config_path.read_text());assert config['version']=='mixed-fit-v1'
    assert sys.version.split()[0]==config['python_version'] and np.__version__==config['numpy_version']
    assert sha(phase1/'rows.jsonl')==config['phase1_rows_sha256']
    rows={r['fixture_id']:r for r in map(json.loads,(phase1/'rows.jsonl').read_text().splitlines())}
    fixtures={r['id']:r for r in map(json.loads,(root/'testdata/mixed-boundary/fixtures.jsonl').read_text().splitlines())}
    excluded={};eligible={}
    for rid,r in rows.items():
        if r['split']!='train' or not r['primary'] or not r['eligible'] or r['intent']=='ambiguous':continue
        if r['ownership']['aggressive'].get('invalid',0) or r['ownership']['aggressive'].get('unsupported',0):
            excluded[rid]='aggressive ownership unavailable';continue
        eligible[rid]=r
    # Keep only train candidates. Test candidates are never used by fitting.
    events={}
    with (phase1/'candidates.jsonl').open() as f:
        for line in f:
            c=json.loads(line);rid=c['fixture_id']
            if rid not in eligible or c['lane']=='opaque' or c['pruned']:continue
            if c['features'] is None or any(x is None for x in c['features']):
                excluded[rid]='null feature excludes whole fixture';continue
            events.setdefault((rid,c['event'],c['window']),[]).append(c)
    for rid in excluded:eligible.pop(rid,None)
    examples={};missing=[]
    for (rid,at,wi),cs in events.items():
        if rid not in eligible:continue
        finals=[a for a in fixtures[rid]['assertions'] if a['policy']=='generator' and a['stage']=='final' and a['window']==wi]
        if not finals:raise ValueError('no terminal gold')
        end=max(s['end'] for s in cs[0]['segments'])
        gold={project(p,end) for a in finals for p in a['accepted']}
        good=[c for c in cs if identity(c['segments']) in gold]
        bad=[c for c in cs if identity(c['segments']) not in gold]
        if not good:
            missing.append({'fixture_id':rid,'event':at,'window':wi});continue
        if not bad:continue
        good.sort(key=lambda c:identity(c['segments']).encode())
        bad.sort(key=lambda c:identity(c['segments']).encode())
        x=np.array(good[0]['features'],dtype=np.float64);x[12]=0
        examples.setdefault(rid,[]).append([(x-np.array(c['features'],dtype=np.float64)).tolist() for c in bad])
    counts={k:sum(rows[r]['intent']==k for r in examples) for k in ['chinese','literal']}
    if not all(counts.values()):raise ValueError('missing training intent')
    matrix=[];weights=[];pair_order=[]
    order=sorted(examples,key=lambda rid:(hashlib.sha256((config['seed']+'\n'+rid).encode()).hexdigest(),rid))
    for rid in order:
        es=examples[rid];intent=rows[rid]['intent']
        for negatives in es:
            for d in negatives:
                d[12]=0;matrix.append(d);weights.append(.5/counts[intent]/len(es)/len(negatives));pair_order.append(rid)
    x=np.asarray(matrix,dtype=np.float64);q=np.asarray(weights,dtype=np.float64);w=np.zeros(26,dtype=np.float64)
    for iteration in range(200):
        margin=x@w
        derivative=np.exp(-np.logaddexp(0,margin))
        grad=-(x.T@(q*derivative))+.001*w
        grad[12]=0;w-=.05*grad;w[12]=0
        if not np.all(np.isfinite(w)):raise ValueError('nonfinite optimizer')
    out.mkdir(exist_ok=False)
    model={'version':'mixed-linear-v1','feature_version':'mixed-features-v1','coefficients':w.tolist(),'margin':0,'fit_spec_sha256':sha(config_path),'numpy':np.__version__,'python':sys.version,'training_primary_rows':counts,'pairs':len(matrix),'feature12_fixed_zero':True}
    (out/'model.json').write_text(json.dumps(model,sort_keys=True,indent=2)+'\n')
    evidence={'config_sha256':sha(config_path),'source_rows_sha256':sha(phase1/'rows.jsonl'),'source_candidates_sha256':sha(phase1/'candidates.jsonl'),'training_primary_rows':counts,'eligible_rows_before_reachable_pair_filter':len(eligible),'excluded_rows':excluded,'missing_gold_events':missing,'pairs':len(matrix),'iterations':200,'no_test_training':True,'model_sha256':sha(out/'model.json')}
    (out/'training.json').write_text(json.dumps(evidence,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    print(json.dumps({'model_sha256':evidence['model_sha256'],'rows':counts,'pairs':len(matrix),'excluded':len(excluded),'missing_gold_events':len(missing)}))

def select(root,trained,dev,out):
    config=json.loads((root/'testdata/mixed-boundary/fitting-v1.json').read_text())
    choices=[];hashes={};denominators=None
    for i,margin in enumerate(config['margin_grid']):
        path=dev/f'dev-margin-{i:02d}'/'report.json';r=json.loads(path.read_text())
        if r['split']!='dev' or r['margin']!=margin:raise ValueError('undeclared development replay')
        if r['fit_spec_sha256']!=sha(root/'testdata/mixed-boundary/fitting-v1.json'):raise ValueError('spec changed')
        ds=tuple(r[k]['denominator'] for k in ['literal_any_event_conversion','harmful_revision','chinese_first_boundary_error','unnecessary_revision'])
        if denominators is None:denominators=ds
        if ds!=denominators or not all(ds):raise ValueError('inconsistent dev denominator')
        key=tuple(r[k]['value'] for k in ['literal_any_event_conversion','harmful_revision','chinese_first_boundary_error','unnecessary_revision'])+(margin,)
        choices.append((key,i,margin));hashes[str(margin)]=sha(path)
    key,i,margin=min(choices)
    model=json.loads((trained/'model.json').read_text());model['margin']=margin
    out.mkdir(exist_ok=False)
    (out/'model.json').write_text(json.dumps(model,sort_keys=True,indent=2)+'\n')
    evidence={'version':'mixed-selected-v1','margin':margin,'dev_key':key,'source_model_sha256':sha(trained/'model.json'),'dev_report_sha256':hashes,'selected_model_sha256':sha(out/'model.json'),'test_scores_seen':False}
    (out/'selection.json').write_text(json.dumps(evidence,sort_keys=True,indent=2)+'\n')
    print(json.dumps(evidence))

if __name__=='__main__':
    if len(sys.argv)==5 and sys.argv[1]=='train':
        train(pathlib.Path(sys.argv[2]),pathlib.Path(sys.argv[3]),pathlib.Path(sys.argv[4]))
    elif len(sys.argv)==6 and sys.argv[1]=='select':
        select(*(pathlib.Path(x) for x in sys.argv[2:]))
    else:
        raise SystemExit('usage: fit.py train REPO PHASE1_OUT NEW_MODEL_OUT | select REPO TRAINED DEV_ROOT NEW_SELECTED_OUT')
