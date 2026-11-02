#!/usr/bin/env python3
"""Frozen v2 full-batch pairwise linear fitting; no task generation or test access."""
# Single-thread settings precede NumPy import, even when invoked directly.
import os
for _variable in ('OPENBLAS_NUM_THREADS', 'OMP_NUM_THREADS', 'MKL_NUM_THREADS', 'NUMEXPR_NUM_THREADS'):
    os.environ[_variable] = '1'
import argparse
import math
from pathlib import Path
import sys
import numpy as np
from exchange import (ROOT, Insufficient, digest, encoded, filehash, newout,
                      parse, record, records, verify_lock, write)

FEATURES = ['literal', 'english_prefix_known', 'prefix_has_digit', 'prefix_has_punct',
            'prefix_numberlike', 'first_prefix_log', 'first_missing',
            'later_complete_syllables', 'tail_bytes', 'first_word_length',
            'multiword_coverage', 'complete_mean_log', 'complete_missing']
CONTINUOUS = {5, 7, 8, 9, 10, 11}
MASKS = {'E0': list(range(7)), 'EL': list(range(9)),
         'EW': list(range(7)) + list(range(9, 13)), 'EC': list(range(13))}
MARGINS = [0., .1, .25, .5, 1., 2., 4., 8.]

def observation_order(o):
    return (0 if o['_intent'] == 'literal' else 1, o['component_id'],
            o['fixture_id'], o['event_index'], o['candidate']['identity'])

def population(observations):
    means, stds, counts = [0.] * 13, [1.] * 13, [0] * 13
    for j in sorted(CONTINUOUS):
        values = [float(o['candidate']['features'][j]) for o in observations
                  if not (j == 5 and o['candidate']['features'][6] == 1 or
                          j == 11 and o['candidate']['features'][12] == 1)]
        counts[j] = len(values)
        total = 0.
        for value in values:
            total += value  # CPython3.12 sum() uses compensation; contract forbids it.
        means[j] = total / len(values) if values else 0.
        squared = 0.
        for value in values:
            delta = value-means[j]
            squared += delta*delta
        stds[j] = math.sqrt(squared / len(values)) if values else 0.
    return means, stds, counts

def standardized(features, means, stds):
    z = np.array(features, dtype=np.float64)
    for j in CONTINUOUS:
        missing = j == 5 and features[6] == 1 or j == 11 and features[12] == 1
        z[j] = 0. if missing or stds[j] == 0 else (float(features[j])-means[j])/stds[j]
    return z

def build_pairs(observations, means, stds):
    groups = {}
    for o in observations:
        key = (o['_intent'], o['component_id'], o['fixture_id'], o['event_index'])
        groups.setdefault(key, []).append(o)
    valid, diagnostics = {}, {'checkpoints': len(groups), 'no_pair_checkpoints': 0,
                              'primary_rows': len({o['fixture_id'] for o in observations})}
    for key, candidates in groups.items():
        gold_ids = set(candidates[0]['gold_ids'])
        if any(set(c['gold_ids']) != gold_ids or c['raw'] != candidates[0]['raw'] for c in candidates):
            raise ValueError('inconsistent gold/raw within checkpoint')
        if len({c['candidate']['id'] for c in candidates}) != len(candidates):
            raise ValueError('duplicate candidate within checkpoint')
        gold = [c for c in candidates if c['candidate']['id'] in gold_ids]
        negative = [c for c in candidates if c['candidate']['id'] not in gold_ids]
        if gold and negative:
            valid[key] = (gold, negative)
        else:
            diagnostics['no_pair_checkpoints'] += 1
    intents = sorted({k[0] for k in valid}, key=lambda x: 0 if x == 'literal' else 1)
    if not intents:
        raise Insufficient('no valid intent/pair set; zero model forbidden')
    components = {intent: {k[1] for k in valid if k[0] == intent} for intent in intents}
    checkpoints = {(intent, component): sum(k[:2] == (intent, component) for k in valid)
                   for intent in intents for component in components[intent]}
    pair_count = sum(len(g)*len(n) for g, n in valid.values())
    if pair_count > 2000000:
        raise Insufficient(f'{pair_count} pairs exceeds fixed 2000000 budget; no negative sampling')
    pairs = []
    for key in sorted(valid, key=lambda k: (0 if k[0] == 'literal' else 1, *k[1:])):
        gold, negative = valid[key]
        q = 1./len(intents)/len(components[key[0]])/checkpoints[key[:2]]/len(gold)/len(negative)
        for g in sorted(gold, key=lambda c: c['candidate']['identity']):
            for n in sorted(negative, key=lambda c: c['candidate']['identity']):
                difference = standardized(g['candidate']['features'], means, stds)-standardized(n['candidate']['features'], means, stds)
                pairs.append((difference, q))
    no_pair_rows = sorted({o['fixture_id'] for o in observations} - {k[2] for k in valid})
    no_pair_components = sorted({(o['_intent'], o['component_id']) for o in observations} - {k[:2] for k in valid})
    diagnostics.update({'pairs': pair_count, 'effective_intents': intents,
                        'no_pair_rows': len(no_pair_rows), 'no_pair_row_ids': no_pair_rows,
                        'no_pair_components': len(no_pair_components),
                        'no_pair_component_intents': [list(x) for x in no_pair_components],
                        'no_pair_checkpoint_ids': [list(k) for k in groups if k not in valid]})
    return pairs, diagnostics

def objective(weights, pairs, active):
    gradient = np.zeros(13, dtype=np.float64)
    loss = 0.
    for difference, q in pairs:
        score = 0.
        for j in range(13):
            score += float(weights[j])*float(difference[j])
        loss += q * (max(-score, 0.) + math.log1p(math.exp(-abs(score))))
        if score >= 0:
            exp = math.exp(-score)
            sigmoid = exp/(1.+exp)
        else:
            sigmoid = 1./(1.+math.exp(score))
        # Every dimension accumulates in canonical pair order; no BLAS reduction.
        gradient += (-q*sigmoid)*difference
    for j in active:
        loss += .001/2*float(weights[j])**2
        gradient[j] += .001*weights[j]
    for j in set(range(13))-set(active):
        gradient[j] = 0.
    if not math.isfinite(loss) or not np.all(np.isfinite(gradient)):
        raise ValueError('nonfinite objective/gradient')
    return loss, gradient

def optimize(pairs, active):
    weights = np.zeros(13, dtype=np.float64)
    for _ in range(1000):
        _, gradient = objective(weights, pairs, active)
        weights = weights-.05*gradient  # simultaneous update after all components
    loss, gradient = objective(weights, pairs, active)
    squared_norm = 0.
    for value in gradient:
        squared_norm += float(value)*float(value)
    return weights.tolist(), loss, float(math.sqrt(squared_norm))

def fit(args):
    lock = verify_lock(args.lock, (args.observations, args.assignment, __file__, Path(__file__).with_name('exchange.py')))
    if lock['demo']:
        raise ValueError('formal fit rejects demo locks')
    source_locks=[p for p in lock['files'] if Path(p).name=='run-lock.json']
    if len(source_locks)!=1:raise ValueError('fit lock must contain one immutable initial run-lock.json')
    code_lock_hash=filehash(source_locks[0])
    recall_reports=[]
    recall_paths=[]
    for path in lock['files']:
        if Path(path).name=='report.json':
            candidate=record(path,'report')
            if candidate['gates']['G1']['status']=='pass' and candidate['qualification']['source_complete']:recall_reports.append(candidate);recall_paths.append(path)
    if len(recall_reports)!=1:raise Insufficient('fit requires one frozen qualified G1-passing enumeration report')
    enumeration_meta=Path(recall_paths[0]).with_name('run.json')
    if str(enumeration_meta.resolve()) not in {str(Path(p).resolve()) for p in lock['files']}:raise Insufficient('enumeration completion status not frozen')
    phase=parse(enumeration_meta.read_bytes())
    if phase.get('command')!='enumerate' or phase.get('support_complete') is not True:raise Insufficient('enumeration contains required unsupported/exhausted episodes')
    if phase.get('lock_sha256')!=code_lock_hash or phase.get('assignment_sha256')!=filehash(args.assignment):raise ValueError('enumeration source/assignment provenance differs')
    q=recall_reports[0]['qualification']
    if q['test_return_components']<400 or q['test_literal_components']<400 or q['test_english_components']<400 or q['contrast_components']<100:raise Insufficient('frozen enumeration sample qualification insufficient')
    assignment = record(args.assignment, 'assignment')
    primary = {}
    membership = {}
    for component in assignment['components']:
        for fixture in component['row_ids']:
            membership[fixture] = component
        for intent in ('literal', 'return'):
            fixture = component['primary_'+intent]
            if fixture is not None:
                primary[fixture] = intent
    observations = []
    for o in records(args.observations, 'training_observation'):
        component = membership.get(o['fixture_id'])
        if component is None or component['id'] != o['component_id']:
            raise ValueError('observation/assignment join differs')
        if component['split'] != 'train':
            raise ValueError('fit received non-train observations')
        if o['demo']:
            raise ValueError('formal fit rejects demo observations')
        if o['fixture_id'] in primary:
            o['_intent'] = primary[o['fixture_id']]
            observations.append(o)
    observations.sort(key=observation_order)
    means, stds, counts = population(observations)
    pairs, diagnostics = build_pairs(observations, means, stds)
    # Create output before expensive fitting; preserve partial failure evidence.
    out = newout(args.out)
    write(out/'coverage.json', diagnostics)
    dev_models={}
    for name, active in MASKS.items():
        coefficients, loss, norm = optimize(pairs, active)
        model = {'version': 'shift-return-v2', 'kind': 'model', 'demo': False,
                 'name': name, 'feature_names': FEATURES, 'active': active,
                 'coefficients': coefficients, 'means': means, 'stds': stds,
                 'counts': counts, 'margin': 0., 'selected_on': 'dev',
                 'code_lock_sha256': code_lock_hash, 'fit_lock_sha256': filehash(args.lock),
                 'training_loss': loss, 'gradient_l2': norm}
        write(out/(name+'.json'), model)
        dev_models[name]={}
        for margin in MARGINS:
            variant=dict(model,margin=float(margin))
            filename=name+'-margin-'+format(margin,'g')+'.json'
            write(out/filename,variant)
            dev_models[name][format(margin,'g')]=filehash(out/filename)
    write(out/'training-manifest.json', {'assignment': str(Path(args.assignment).resolve()),
          'assignment_sha256': filehash(args.assignment), 'observations_sha256': filehash(args.observations),
          'fit_lock': str(Path(args.lock).resolve()), 'fit_lock_sha256': filehash(args.lock),
          'models': {name: filehash(out/(name+'.json')) for name in MASKS},'dev_models':dev_models})


def select(args):
    train = parse((Path(args.models)/'training-manifest.json').read_bytes())
    if filehash(train['assignment']) != train['assignment_sha256']:
        raise ValueError('training assignment changed')
    selected = {}
    for name in MASKS:
        model_path = Path(args.models)/(name+'.json')
        if filehash(model_path) != train['models'][name]:
            raise ValueError('trained model changed')
        candidates = []
        for margin in MARGINS:
            directory = Path(args.dev)/(name+'-margin-'+format(margin, 'g'))
            meta = parse((directory/'run.json').read_bytes())
            if meta['split'] != 'dev' or meta['flow'] != 'revise' or meta['model_sha256'] != train['dev_models'][name][format(margin,'g')] or float(meta['margin']) != margin:
                raise ValueError('selection input is not the frozen dev-revise model/margin')
            report = record(directory/'report.json', 'report')
            metrics = report['conditions'][name+'-revise']
            if any(metrics[k]['n'] == 0 or metrics[k]['value'] is None for k in ('return_boundary', 'literal_only', 'return_english')):
                raise Insufficient('dev denominator/evidence unavailable')
            ret, lit, eng = (metrics[k]['value'] for k in ('return_boundary', 'literal_only', 'return_english'))
            feasible = ret <= .1 and lit <= .01 and eng <= .01
            candidates.append((feasible, ret, lit, eng, margin))
        feasible = [x for x in candidates if x[0]]
        chosen = min(feasible, key=lambda x: (x[1], x[2], x[3], x[4])) if feasible else min(candidates, key=lambda x: (max(x[2], x[3]), x[2]+x[3], x[1], x[4]))
        model = record(model_path, 'model')
        model['margin'] = float(chosen[4])
        selected[name] = (model, bool(feasible))
    out = newout(args.out)
    for name, (model, _) in selected.items():
        write(out/(name+'.json'), model)
    source_hashes={model['code_lock_sha256'] for model,_ in selected.values()}
    fit_hashes={model['fit_lock_sha256'] for model,_ in selected.values()}
    if len(source_hashes)!=1 or len(fit_hashes)!=1:raise ValueError('selected models have different source/fit ancestry')
    write(out/'selected-model-manifest.json',{'version':'shift-return-v2','kind':'selected_model_manifest','demo':False,'source_lock_sha256':source_hashes.pop(),'fit_lock_sha256':fit_hashes.pop(),'models':{name:{'path':str((out/(name+'.json')).resolve()),'sha256':filehash(out/(name+'.json'))} for name in MASKS}})
    write(out/'selection.json', {name: {'dev_passed': feasible, 'margin': model['margin'],
          'model_sha256': filehash(out/(name+'.json'))} for name, (model, feasible) in selected.items()})


def verify_numerical(path):
    n = parse(Path(path).read_bytes())
    close = lambda a, b: math.isclose(a, b, rel_tol=0, abs_tol=1e-12)
    observations = [{'_intent': o['intent'], 'component_id': o['component_id'],
        'fixture_id': o['fixture_id'], 'event_index': o['event_index'],
        'candidate': {'identity': o['candidate_identity'], 'features': [0.]*5+[o['x']]+[0.]*7}}
        for o in n['ordered_observations']]
    observations.sort(key=observation_order)
    means, stds, _ = population(observations)
    assert close(means[5], n['mean']) and close(stds[5], n['std'])
    d = np.zeros(13); d[5] = n['pair_difference']
    pairs = [(d, n['pair_weight'])]
    w = np.zeros(13)
    loss, gradient = objective(w, pairs, [5])
    assert close(loss, n['initial_loss']) and close(gradient[5], n['initial_gradient'])
    assert close(float((w-.05*gradient)[5]), n['after_one_update'])
    w[5] = n['probe_w']; loss, gradient = objective(w, pairs, [5])
    assert close(loss, n['probe_loss']) and close(gradient[5], n['probe_gradient'])
    assert encoded({'w': -0.0}) == b'{"w":0.0}'
    assert digest(record(ROOT/'design/shift-return-v2/canonical-model-sample.json', 'model')) == n['canonical_model_sha256']
    first, second = optimize(pairs, [5]), optimize(pairs, [5])
    assert encoded(first) == encoded(second)
    print('Numerical verification passed: ordered population, loss/gradient/update, canonical hash, two byte-identical tiny fits; no quality fitting.')


def main():
    parser = argparse.ArgumentParser()
    sub = parser.add_subparsers(dest='command', required=True)
    p = sub.add_parser('fit')
    for name in ('observations', 'assignment', 'lock', 'out'):
        p.add_argument('--'+name, required=True)
    p = sub.add_parser('select')
    for name in ('dev', 'models', 'out'):
        p.add_argument('--'+name, required=True)
    p = sub.add_parser('verify-numerical'); p.add_argument('example')
    args = parser.parse_args()
    try:
        if args.command == 'fit': fit(args)
        elif args.command == 'select': select(args)
        else: verify_numerical(args.example)
    except Insufficient as error:
        print(error, file=sys.stderr); return 3
    except (ValueError, OSError, KeyError, AssertionError) as error:
        print(error, file=sys.stderr); return 2
    return 0

if __name__ == '__main__':
    sys.exit(main())
