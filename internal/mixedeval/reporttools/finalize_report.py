#!/usr/bin/env python3
"""Adjudicate fixed gates from frozen raw evidence; no training or replay."""
import sys
import collections
import hashlib
import json
import pathlib
import math

B=pathlib.Path(sys.argv[2]).resolve();R=pathlib.Path(sys.argv[1]).resolve()
def read(p):return json.loads(p.read_text())
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for block in iter(lambda:f.read(1<<20),b''):h.update(block)
 return h.hexdigest()
def identity(p):return json.dumps([[s['kind'],s['begin'],s['end'],s.get('reading','')] for s in p],ensure_ascii=False,separators=(',',':'))
def rate(n,d):
 if not d:return {'numerator':n,'denominator':0,'value':None,'wilson95':None}
 p=n/d;z=1.959963984540054;zz=z*z;c=(p+zz/(2*d))/(1+zz/d);h=z*math.sqrt(p*(1-p)/d+zz/(4*d*d))/(1+zz/d)
 return {'numerator':n,'denominator':d,'value':p,'wilson95':[max(0,c-h),min(1,c+h)]}
def gate(status,reason):return {'status':status,'pass':True if status=='pass' else False if status=='fail' else None,'reason':reason}
def main():
 p1=read(B/'phase1-corrected-report.json');test=read(B/'test-v1/report.json');rows=[json.loads(s) for s in (B/'test-v1/rows.jsonl').read_text().splitlines()];fixtures={r['id']:r for r in map(json.loads,(R/'testdata/mixed-boundary/fixtures.jsonl').read_text().splitlines())}
 # Audit own-causal candidate survival separately from the preregistered
 # Phase1 lagged-context gate. No labels are provided to the replay policy.
 candidates=collections.defaultdict(set)
 for line in (B/'test-v1/candidates.jsonl').open():
  c=json.loads(line);x=c['candidate']
  if not x['pruned'] and x['lane'] in ['strict','literal']:candidates[c['fixture_id'],c['event'],c['window']].add(identity(x['segments']))
 causal_bad=[];causal_n=0
 for row in rows:
  if row['intent']!='chinese' or not row['primary']:continue
  causal_n+=1;f=fixtures[row['fixture_id']];ok=True
  for a in f['assertions']:
   if a['policy']=='generator' and a['stage']=='final':ok=ok and any(identity(p) in candidates[f['id'],a['at'],a['window']] for p in a['accepted'])
  if not ok:causal_bad.append(row['fixture_id'])
 categories={};taxonomy={}
 for cat,data in test['categories'].items():categories[cat]=data['test_primary']
 for intent,key in [('chinese','first_boundary_error'),('literal','literal_conversion')]:
  rs=[r for r in rows if r['intent']==intent];bad=[r for r in rs if r[key].get('proposed')]
  taxonomy[intent]={'row_error':rate(len(bad),len(rs)),'worst_case_group_incidence':rate(len({r['group_id'] for r in bad}),len({r['group_id'] for r in rs})),'error_ids':[r['fixture_id'] for r in bad],'primary_error_ids':[r['fixture_id'] for r in bad if r['primary']]}
 prim=[r for r in rows if r['primary'] and r['intent']!='ambiguous'];newly_bad=[r['fixture_id'] for r in prim if r['intent']=='chinese' and r['first_boundary_error'].get('aggressive') is False and r['first_boundary_error'].get('proposed') is True];fixed=[r['fixture_id'] for r in prim if r['intent']=='chinese' and r['first_boundary_error'].get('aggressive') is True and r['first_boundary_error'].get('proposed') is False]
 all_invariants={k:sum(r[k] for r in rows) for k in ['outside_changes','key_failures','commit_failures','key_events','revisions','fixed_fallbacks']}
 gates={
  'G1':gate('unavailable','Executed schema/semantic/split, exhaustive, guard/observer and event subchecks passed; whole evaluator correctness is not established because context presentation omits the frozen prelude from E run-scoped commit.'),
  'G2':gate('pass','Protected generation/scoring zero in executed protection tests; every hypothetical-store digest unchanged, no persistence API; actual baseline commit learning confined to disposable per-row stores.'),
  'G3':gate('pass','Main frozen TEST primary capped strict recall492/492; mandatory10/10; eligible ambiguous150/150 both alternatives. Conditional-on-legacy frozen history limitation is explicit.'),
  'G4':gate('fail','A40/492 versus E178/492 errors; relative error reduction=-3.45, below required+0.30.'),
  'G5':gate('fail','E1/164 literal any-event conversion; point0.6098% but Wilson95 upper3.3726% exceeds1%;164 literal test groups also below400.'),
  'G6':gate('fail','Unnecessary revision10/656=1.5244% exceeds1%; harmful0 and outside-window changes0 do not repair this failure.'),
  'G7':gate('unavailable','Run-owned bytes/Enter/freeze checks zero failures, but E context replay emits only the run and omits frozen prelude visible text from the composition commit. Full original-key/commit preservation is not demonstrated; native editing remains delegated_unverified.'),
  'G8':gate('unavailable','Literal test164<400; sparse release categories,3 baseline literal primary ownership paths unsupported, and declared known-prefix category includes unverified pluralized prefixes. Full necessary paired evidence cannot pass.')}
 result={'version':'mixed-final-adjudication-v1','outcome':'NO_GO','interpretation_scope':'negative result for this preregistered estimator/window/data protocol; no product integration, no universal impossibility claim','gates':gates,'provenance':{'acceptance_commit':'1c927821b7ae7e4863cf8a783dbffc712429167b','acceptance_sha256':sha(R/'design/mixed-input-experiment-acceptance.md'),'phase1_freeze':'169adcffc304d1fa1e8b2b8bc769761248638a0b','prefit_freeze':'1244ee6ba1287562c9c701209d2eb8058e8731e2','numeric_freeze':'9203c4f6aace9c7261ece948fa614fce888976d5','run_lock_sha256':sha(R/'testdata/mixed-boundary/run-lock.json'),'fitting_lock_sha256':sha(R/'testdata/mixed-boundary/fitting-run-lock.json'),'fixture_sha256':sha(R/'testdata/mixed-boundary/fixtures.jsonl'),'assignment_sha256':sha(R/'testdata/mixed-boundary/assignment.json'),'model_sha256':sha(B/'selected-v1/model.json'),'test_rows_sha256':sha(B/'test-v1/rows.jsonl'),'ranked_test_runs':1},'phase1_G3':p1['G3'],'causal_ranked_candidate_recall_diagnostic':{'metric':rate(causal_n-len(causal_bad),causal_n),'failure_ids':causal_bad},'metrics':{k:test[k] for k in ['chinese_first_boundary_error','literal_any_event_conversion','baseline_aggressive_first_boundary_error_mapped','relative_error_reduction','useful_revision','harmful_revision','unnecessary_revision','benign_revision','ownership_revisions','fixed_fallbacks','outside_changes','key_failures','commit_failures']},'baseline':{'aggressive':read(B/'enumerate-v1/report.json')['baseline_diagnostics']['aggressive']['test_primary'],'conservative':read(B/'enumerate-v1/report.json')['baseline_diagnostics']['conservative']['test_primary'],'causal_longest':read(B/'longest-causal-test-v1/report.json'),'corrected_any_event_literal':p1['baseline_test_primary_any_event_literal']},'all_ranked_row_invariants':all_invariants,'categories_declared_before_quality':categories,'failure_taxonomy':taxonomy,'paired_chinese_transitions':{'newly_wrong_count':len(newly_bad),'newly_wrong_ids':newly_bad,'fixed_count':len(fixed),'fixed_ids':fixed},'dataset_qualification':test['qualification'],'train':read(B/'trained-v1/training.json'),'train_diagnostic':read(B/'training-diagnostic.json'),'selection':read(B/'selected-v1/selection.json'),'full_state_exhaustive_additional_after_training':read(B/'full-state-exhaustive.json'),'reporting_corrections':['Original Phase1 automatic G3 used all-row denominator; corrected main TEST primary report preserved separately.','Original Phase1 literal metrics used terminal candidate comparisons; any-event A/C and causal-longest diagnostics preserved separately.','Original frozen reports were never overwritten; no data, feature, cap, coefficient or margin changed after outcomes.'],'limitations':['fixture declares run origin/context; live automatic window discovery is untested','E context presentation and commit are run-scoped; frozen prelude visibility is not materialized, so full-composition G1/G7 cannot pass','synthetic Chinese intentions and public hardware identifiers do not estimate natural user typing','declared known-prefix includes266 pre-quality relabeled pluralized-prefix rows whose exact English membership was not validated; no post-outcome relabeling performed','full test literal count and category support insufficient','homophone/surface gold absent for primary test cohort, so surface-error denominator0 remains null','Windows native TSF/UI/latency and neighbor2x2 remain untested','fixed-budget objective improved but nonzero gradient does not prove optimizer convergence; this one estimator does not rule out every bounded parser or linear ranker']}
 (B/'final-report.json').write_text(json.dumps(result,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
 print(json.dumps({'outcome':result['outcome'],'causal_recall':result['causal_ranked_candidate_recall_diagnostic'],'all_invariants':all_invariants,'new_wrong':len(newly_bad),'fixed':len(fixed)}))
if __name__=='__main__':main()
