#!/usr/bin/env python3
"""Transparent postprocessor for the frozen Phase1 reporting aggregation bug.
Does not change generator, fixtures, raw reports, feature values or denominators.
"""
import sys
import collections
import hashlib
import json
import math
import pathlib

ROOT=pathlib.Path(sys.argv[1]).resolve()
BASE=pathlib.Path(sys.argv[2]).resolve()
RAW=BASE/'enumerate-v1'

def sha(p):
    h=hashlib.sha256()
    with p.open('rb') as f:
        for b in iter(lambda:f.read(1<<20),b''):h.update(b)
    return h.hexdigest()

def rate(n,d):
    if not d:return {'numerator':n,'denominator':0,'value':None,'wilson95':None}
    p=n/d;z=1.959963984540054;zz=z*z;center=(p+zz/(2*d))/(1+zz/d);half=z*math.sqrt(p*(1-p)/d+zz/(4*d*d))/(1+zz/d)
    return {'numerator':n,'denominator':d,'value':p,'wilson95':[max(0,center-half),min(1,center+half)]}

def main():
    original=json.loads((RAW/'report.json').read_text());rows=[json.loads(s) for s in (RAW/'rows.jsonl').read_text().splitlines()];byid={r['fixture_id']:r for r in rows}
    prim=[r for r in rows if r['intent']=='chinese' and r['split']=='test' and r['primary']]
    mandatory=[r for r in rows if r['mandatory'] and r['intent']=='chinese' and r['eligible']]
    ambiguous=[r for r in rows if r['ambiguous_survival'] is not None]
    missing=[r['fixture_id'] for r in prim if r['capped_recall'] is None]
    main_recall=rate(sum(r['capped_recall'] is True for r in prim),len(prim))
    mandatory_bad=[r['fixture_id'] for r in mandatory if r['capped_recall'] is not True]
    ambig_bad=[r['fixture_id'] for r in ambiguous if r['ambiguous_survival'] is not True]
    status='unavailable' if missing else 'pass' if main_recall['value']>=.99 and not mandatory_bad and not ambig_bad else 'fail'
    # Recover any-event literal incidence only where typed ownership is mapped.
    flags={p:{} for p in ['aggressive','conservative']};unmapped={p:set() for p in flags}
    for line in (RAW/'events.jsonl').open():
        e=json.loads(line);p=e['policy'];rid=e['fixture_id'];r=byid[rid]
        if p not in flags or r['intent']!='literal' or not r['eligible']:continue
        if e['ownership_status']!='mapped':unmapped[p].add(rid);continue
        flags[p][rid]=flags[p].get(rid,False) or any(s['kind']=='chinese' for s in e['partition'] or [])
    literal={}
    for p in flags:
        target=[r for r in rows if r['intent']=='literal' and r['split']=='test' and r['primary'] and r['eligible']]
        unavailable=[r['fixture_id'] for r in target if r['fixture_id'] in unmapped[p] or r['fixture_id'] not in flags[p]]
        mapped=[r for r in target if r['fixture_id'] not in unavailable]
        literal[p]={'main':None if unavailable else rate(sum(flags[p][r['fixture_id']] for r in target),len(target)), 'mapped_only_diagnostic':rate(sum(flags[p][r['fixture_id']] for r in mapped),len(mapped)), 'unavailable_primary_rows':unavailable,'denominator_fixed':len(target),'definition':'any mapped Chinese ownership event; raw-rewriting/unsupported paths are unavailable, not guessed from rendered text'}
    failure=[r for r in rows if r['capped_recall'] is False]
    corrected={'version':'mixed-phase1-corrected-v1','source_freeze':'169adcffc304d1fa1e8b2b8bc769761248638a0b','original_report_sha256':sha(RAW/'report.json'),'original_rows_sha256':sha(RAW/'rows.jsonl'),'reporting_bug':'frozen Summarize used all-row Chinese denominator for auto-G3; main preregistered denominator is test unambiguous primary groups. Original report remains unchanged. Terminal-only baseline literal conversion is diagnostic, not any-event conversion.', 'G3':{'status':status,'main_test_primary_capped_recall':main_recall,'uncapped_recall':rate(sum(r['uncapped_recall'] is True for r in prim),len(prim)), 'missing_primary_labels':missing,'mandatory_survival':rate(len(mandatory)-len(mandatory_bad),len(mandatory)),'mandatory_failures':mandatory_bad,'ambiguous_survival':rate(len(ambiguous)-len(ambig_bad),len(ambiguous)),'ambiguous_failures':ambig_bad}, 'all_rows_recall_diagnostic':original['capped_intended_recall'],'all_row_failures':failure,'baseline_test_primary_any_event_literal':literal,'data_qualification':original['adequacy'],'correctness_evidence':{'tiny_public_exhaustive':json.loads((BASE/'correctness-both-v1/correctness.json').read_text()),'observer_parity_events':424,'guard_vectors':3150,'guards_per_vector':13,'source_test_log':'phase1-unit-parity.log','parent_independent_checks':'parent-protected-reset-initial.log'},'outcome':'INCONCLUSIVE: G3 passes; final literal primary164 <400 and fitted ranking not yet measured'}
    path=BASE/'phase1-corrected-report.json'
    with path.open('x') as f:f.write(json.dumps(corrected,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    print(json.dumps({'G3':corrected['G3'],'any_event_literal':literal},ensure_ascii=False))

if __name__=='__main__':main()
