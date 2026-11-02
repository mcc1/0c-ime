#!/usr/bin/env python3
"""Independent frozen-evidence reducer: joins/grades/counts; never selects or replays."""
import argparse
import math
from pathlib import Path
import random
import sys
import unicodedata
from exchange import (Insufficient, VALIDATOR, digest, encoded, filehash, newout,
                      normalized_prompt, parse, record, records, verify_lock, write, validate_metric)
from dictionary import public_dictionary

CONDITIONS = [name+'-'+flow for name in ('E0', 'EL', 'EW', 'EC') for flow in ('fixed', 'revise')]
METRICS = ('return_boundary', 'return_surface', 'literal_only', 'return_english')
SAFETY = ('key_failures', 'commit_failures', 'app_rewrites', 'lock_violations',
          'hypothetical_learning_updates', 'persistent_writes', 'protected_calls')
KEYS = dict(pair for pair in ('1ㄅ','qㄆ','aㄇ','zㄈ','2ㄉ','wㄊ','sㄋ','xㄌ','eㄍ','dㄎ','cㄏ',
    'rㄐ','fㄑ','vㄒ','5ㄓ','tㄔ','gㄕ','bㄖ','yㄗ','hㄘ','nㄙ','uㄧ','jㄨ','mㄩ','8ㄚ','iㄛ','kㄜ',
    ',ㄝ','9ㄞ','oㄟ','lㄠ','.ㄡ','0ㄢ','pㄣ',';ㄤ','/ㄥ','-ㄦ',' ˉ','6ˊ','3ˇ','4ˋ','7˙'))

def identity(segments):
    return encoded([[s['kind'], s['begin'], s['end'], s['reading']] for s in segments]).decode()

def hid(segments):
    import hashlib
    return 'h:'+hashlib.sha256(identity(segments).encode()).hexdigest()

def query_id(fixture, condition, event, raw):
    return 'q:'+digest({'fixture_id':fixture,'condition':condition,'event_index':event,'raw':raw})

def fixture_semantics(f):
    raw = ''.join(e['key'] for e in f['events'] if e['type']=='key')
    if raw != f['expected_raw']:
        raise ValueError('fixture raw/event differs')
    if f['intent'] != 'compat' and (len(f['events'])<3 or f['events'][0]['type']!='shift' or f['events'][-1]['type']!='enter' or any(e['type']!='key' for e in f['events'][1:-1])):
        raise ValueError('formal episode shape differs')
    for gold in f['gold']:
        if f['intent']!='compat' and sum(s['kind']=='literal' for s in gold['segments'])>1:raise ValueError('formal literal prefix split into spans')
        end=0; chinese=0
        for s in gold['segments']:
            if s['begin']!=end or not end<s['end']<=len(raw):
                raise ValueError('gold ownership gap/overlap/range')
            end=s['end']
            if s['kind']=='chinese':
                chinese+=1
                mapped=''.join(KEYS[k] for k in raw[s['begin']:s['end']])
                if mapped.endswith('ˉ'):mapped=mapped[:-1]
                if mapped!=s['reading']:raise ValueError('gold raw/reading differs')
            elif s['reading']!='':raise ValueError('literal/pending reading nonempty')
            if s['kind']=='pending' and f['intent']!='compat':raise ValueError('formal gold pending')
        if end!=len(raw) or (f['intent']=='return' and not chinese) or (f['intent']=='literal' and chinese):
            raise ValueError('gold intent/coverage differs')
        if f['intent']=='literal' and any(x!=raw for x in gold['accepted_surfaces']):raise ValueError('literal gold rewrites raw')
        if f['intent']=='return':
            begin=next(s['begin'] for s in gold['segments'] if s['kind']=='chinese')
            if any(not surface.startswith(raw[:begin]) for surface in gold['accepted_surfaces']):raise ValueError('gold surface rewrites literal prefix')

def validate_assignment(fixtures,sources,assignment,fixtures_path):
    if assignment['fixtures_sha256']!=filehash(fixtures_path):raise ValueError('fixture file hash differs')
    seen={k:sorted(set(sources[k])) for k in ('seen_prefixes','seen_contents','seen_raw')}
    if assignment['seen_sha256']!=digest(seen):raise ValueError('seen hash differs')
    normalized_prefixes={x.lower() for x in seen['seen_prefixes']};normalized_contents={unicodedata.normalize('NFC',x) for x in seen['seen_contents']};normalized_readings={' '.join(x.split()) for x in seen['seen_contents']}
    fs={f['id']:f for f in fixtures}
    if len(fs)!=len(fixtures):raise ValueError('duplicate fixture ID')
    parent={f['id']:f['id'] for f in fixtures}
    def root(i):
        while parent[i]!=i:
            parent[i]=parent[parent[i]];i=parent[i]
        return i
    owners={}
    for f in fixtures:
        keys=[('prefix:'+k[7:].lower()) if k.startswith('prefix:') else ('word:'+unicodedata.normalize('NFC',k[5:])) if k.startswith('word:') else ('word:'+unicodedata.normalize('NFC',k[8:])) if k.startswith('content:') else ('reading:'+' '.join(k[8:].split())) if k.startswith('reading:') else k for k in f['families']]+['raw:'+f['expected_raw']]+(['literal:'+f['expected_raw']] if f['intent']=='literal' else [])+['prototype:'+f['prototype_id'],'observable:'+encoded([f['prelude_events'],f['expected_prelude'],f['expected_raw']]).decode()]
        for gold in f['gold']:
            chinese=[s for s in gold['segments'] if s['kind']=='chinese']
            if chinese:
                prefix=f['expected_raw'][:chinese[0]['begin']]
                if prefix:keys.append('prefix:'+prefix.lower())
                keys.append('reading:'+' '.join(s['reading'] for s in chinese))
                keys.extend('word:'+unicodedata.normalize('NFC',surface)[len(prefix):] for surface in gold['accepted_surfaces'] if unicodedata.normalize('NFC',surface).startswith(prefix))
        for key in keys:
            if key in owners:parent[root(f['id'])]=root(owners[key])
            else:owners[key]=f['id']
    groups={}
    for f in fixtures:groups.setdefault(root(f['id']),[]).append(f)
    expected=[]
    for members in groups.values():
        ids=sorted(f['id'] for f in members);component=digest(ids)
        import hashlib
        bucket=int(hashlib.sha256(('shift-return-v2:'+component).encode()).hexdigest(),16)%10
        split='train' if bucket<6 else 'dev' if bucket<8 else 'test'
        for f in members:
            if f['expected_raw'] in seen['seen_raw']:split='regression'
            for family in f['families']:
                if family.startswith('prefix:') and family[7:].lower() in normalized_prefixes:split='regression'
                if family.startswith('word:') and unicodedata.normalize('NFC',family[5:]) in normalized_contents:split='regression'
                if family.startswith('content:') and unicodedata.normalize('NFC',family[8:]) in normalized_contents:split='regression'
                if family.startswith('reading:') and ' '.join(family[8:].split()) in normalized_readings:split='regression'
                if family.startswith('literal:') and family[8:] in seen['seen_raw']:split='regression'
                if family.startswith('raw:') and family[4:] in seen['seen_raw']:split='regression'

            for gold in f['gold']:
                prefix='';readings=[]
                for s in gold['segments']:
                    if s['kind']=='literal' and not readings:prefix+=f['expected_raw'][s['begin']:s['end']]
                    if s['kind']=='chinese':readings.append(s['reading'])
                content=[' '.join(readings)]+[s[len(prefix):] for s in gold['accepted_surfaces'] if s.startswith(prefix)] if readings else []
                if prefix.lower() in normalized_prefixes or any(unicodedata.normalize('NFC',x) in normalized_contents for x in content) or bool(readings) and ' '.join(readings) in normalized_readings:split='regression'
        c={'id':component,'row_ids':ids,'split':split,'categories':sorted({f['category'] for f in members})}
        for intent in ('return','literal'):
            eligible=sorted((f for f in members if f['intent']==intent),key=lambda f:(f['source_priority'],f['id']))
            primary=eligible[0] if eligible else None
            c['primary_'+intent]=primary['id'] if primary else None
            c['primary_'+intent+'_metadata']=({'id':primary['id'],'category':primary['category'],'stratum':primary['stratum'],'context_class':primary['context_class'],'contrast_eligible':intent=='return' and primary['context_class']=='contrast'} if primary else None)
        expected.append(c)
    if sorted(expected,key=lambda c:c['id'])!=assignment['components']:raise ValueError('deterministic component/split/primary assignment differs')
    return fs

def source_qualification(fixtures,sources,bank,lock):
    if sources['demo'] or any(f['demo'] for f in fixtures):return False
    directories=[Path(path).parent for path in lock['files'] if Path(path).name=='manifest.json' and (Path(path).parent/'english/manifest.json').exists()]
    if not directories:return False
    entries,english=public_dictionary(directories[0],{str(Path(p).resolve()) for p in lock['files']})
    exact={r for _,r in entries}
    source_by_id={s['id']:s for s in sources['entries']}
    if len(source_by_id)!=len(sources['entries']):raise ValueError('duplicate source ID')
    for s in sources['entries']:
        if not all(s[k] for k in ('license','reference','version','cached_path')) or len(set(s['annotator_ids']))!=2:return False
        if filehash(s['cached_path'])!=s['sha256']:raise ValueError('source cache hash differs')
        if s['kind']=='synthetic-intent' and list(records(s['cached_path'],'bank_task'))!=bank:raise ValueError('synthetic source cache is not frozen bank')
        if s['adjudication_log'] and str(Path(s['adjudication_log']).resolve()) not in {str(Path(x).resolve()) for x in lock['files']}:raise ValueError('adjudication log not frozen in lock')
    tasks={t['id']:t for t in bank}
    if len(tasks)!=len(bank):raise ValueError('duplicate bank task ID')
    used=set()
    for f in fixtures:
        if f['intent']=='compat':continue
        task=tasks.get(f['id'].removeprefix('task:'))
        if task is None or f['id']!='task:'+task['id']:return False
        used.add(task['id'])
        if task['demo'] or normalized_prompt(task['prompt'])!=task['prompt']:raise ValueError('bank demo/noncanonical prompt')
        source=source_by_id.get(task['source_ref'])
        if source is None:return False
        label=digest({k:task[k] for k in ('intent','raw','gold')})
        anns=task['annotations']
        if len({a['annotator_id'] for a in anns})!=2 or any(not a['accepted'] or not a['reading_verified'] or a['intent']!=task['intent'] or a['labels_sha256']!=label for a in anns):return False
        if {a['annotator_id'] for a in anns}!=set(source['annotator_ids']):raise ValueError('source/annotation IDs differ')
        if any(a['adjudication_ref'] for a in anns) and not source['adjudication_log']:return False
        if task['variant_index']==0:
            proto=digest({k:task[k] for k in ('source_ref','intent','prompt','raw','gold')})[:24]
            if task['id']!=proto or task['prototype_id']!=proto:raise ValueError('prototype hash differs')
        elif task['prototype_id'] not in tasks or tasks[task['prototype_id']]['variant_index']!=0:raise ValueError('variant prototype missing')
        copy={'version':'shift-return-v2','kind':'fixture','demo':task['demo'],'id':'task:'+task['id'],
              **{k:task[k] for k in ('prototype_id','intent','category','stratum','context_class','source_ref','source_priority','families','prelude_events','expected_prelude','gold')},
              'expected_raw':task['raw'],'events':[{'type':'shift'}]+[{'type':'key','key':k} for k in task['raw']]+[{'type':'enter'}]}
        if copy!=f:raise ValueError('bank/fixture labels differ')
        if bool(task['designated_typo_spans'])!=(task['category']=='typo'):raise ValueError('typo designation differs')
        chinese=[s for s in task['gold'][0]['segments'] if s['kind']=='chinese']
        if task['intent']=='return':
            begin,end=chinese[0]['begin'],chinese[0]['end'];prefix=task['raw'][:begin];n=len(chinese)
            stratum='1' if n==1 else '2-4' if n<=4 else '5-16' if n<=16 else '17-64'
            category='known' if prefix.lower() in english else 'unknown'
            if any('A'<=c<='Z' for c in prefix):category='uppercase'
            if task['raw'][begin] in '012589':category='digit_initial'
            if any(not c.isascii() or not c.isalpha() for c in prefix):category='identifier'
            if task['designated_typo_spans']:category='typo'
            if category!=task['category'] or stratum!=task['stratum'] or n>64 or len(task['raw'])>512:raise ValueError('return category/stratum differs')
            contrast=collision=False
            for other in bank:
                if other['id']==task['id'] or not other['gold']:continue
                oc=[s for s in other['gold'][0]['segments'] if s['kind']=='chinese']
                different=other['intent']=='literal' or (oc[0]['begin'] if oc else -1)!=begin
                if different and other['raw']==task['raw']:collision=True
                if different and other['raw']!=task['raw'] and other['raw'][:end]==task['raw'][:end] and len(other['raw'])>=end and any(tuple(s['reading'] for s in chinese[:length]) in exact for length in range(2,min(8,n)+1)):contrast=True
            context='identical_collision' if collision else 'contrast' if contrast else 'none'
            if context!=task['context_class']:raise ValueError('contrast predicate differs')
        else:
            if task['category'] not in ('ordinary_english','account','phonetic_account','model_version','numeric_date_time','url_path','identifier','random_ascii'):raise ValueError('unknown literal category')
            n=len(task['raw']);stratum='1-8' if n<=8 else '9-32' if n<=32 else '33-128' if n<=128 else '129-512'
            if n>512 or task['stratum']!=stratum:raise ValueError('literal stratum differs')
    if any(all(a['accepted'] for a in t['annotations']) and t['id'] not in used for t in bank):raise ValueError('accepted task omitted from fixtures')
    return True

def wilson(errors,n):
    if not n:return None
    z=1.959963984540054;p=errors/n;den=1+z*z/n
    center=(p+z*z/(2*n))/den;half=z*math.sqrt(p*(1-p)/n+z*z/(4*n*n))/den
    return [max(0.,center-half),min(1.,center+half)]

def metric(values):
    n=len(values);e=sum(values)
    return {'status':'complete' if n else 'unavailable','observed_errors':e,'unavailable_components':0,'errors':e,'n':n,'value':e/n if n else None,'wilson95':wilson(e,n),'independent_components':True}

def quantile(values,p):
    x=(len(values)-1)*p;lo=math.floor(x);hi=math.ceil(x)
    return values[lo]+(values[hi]-values[lo])*(x-lo)

def bootstrap(union,functions):
    union=sorted(union);rng=random.Random(271828);values=[[] for _ in functions]
    if union:
        for _ in range(10000):
            sample=[union[rng.randrange(len(union))] for _ in union]
            for i,function in enumerate(functions):
                value=function(sample)
                if value is not None:values[i].append(value)
    return [( [quantile(sorted(v),.025),quantile(sorted(v),.975)] if len(v)>=9500 else None,len(v)) for v in values]

def paired(baseline,challenger,cohort,base,new,union):
    if set(base)!=set(new):raise ValueError('paired component sets differ')
    repaired=sum(base[k] and not new[k] for k in base);worsened=sum(not base[k] and new[k] for k in base)
    def statistic(sample):
        ids=[i for i in sample if i in base]
        return sum(int(base[i])-int(new[i]) for i in ids)/len(ids) if ids else None
    ci,valid=bootstrap(union,[statistic])[0] if base else (None,0)
    return {'baseline':baseline,'challenger':challenger,'cohort':cohort,'n':len(base),'repaired':repaired,'worsened':worsened,'net':repaired-worsened,'relative_error_reduction':(repaired-worsened)/sum(base.values()) if sum(base.values()) else None,'absolute_error_reduction_ci95':ci,'valid_bootstrap':valid,'discordant_components':repaired+worsened}

def score(model,features):
    total=0.
    for j in range(13):
        x=features[j]
        if j in (5,7,8,9,10,11):
            missing=j==5 and features[6]==1 or j==11 and features[12]==1
            x=0. if missing or model['stds'][j]==0 else (x-model['means'][j])/model['stds'][j]
        total+=model['coefficients'][j]*x
    return total

def grade(f,h):
    boundary=not any(g['segments']==h['display_segments'] for g in f['gold'])
    surface=not any(h['display'] in g['accepted_surfaces'] for g in f['gold'])
    if f['intent']=='return':
        begin=next(s['begin'] for s in f['gold'][0]['segments'] if s['kind']=='chinese')
        english=any(s['kind']=='chinese' and s['begin']<begin for s in h['display_segments'])
        if h['cut']>begin:english=english or not h['display'].startswith(f['expected_raw'][:begin])
    else:english=surface
    return boundary,surface,english

def projected_gold_ids(f,offset):
    ids=set()
    for gold in f['gold']:
        chinese=[s for s in gold['segments'] if s['kind']=='chinese']
        if f['intent']=='literal' or not chinese or offset<chinese[0]['end']:
            segs=[{'kind':'literal','begin':0,'end':offset,'reading':''}] if offset else []
        else:
            segs=[]
            for seg in gold['segments']:
                if seg['begin']>=offset:break
                s=dict(seg)
                if s['end']>offset:
                    s['end']=offset
                    if s['kind']=='chinese':s['kind']='pending';s['reading']=''
                segs.append(s)
                if seg['end']>=offset:break
        ids.add(hid(segs))
    return ids

def temporal(f,episode,candidates,generators):
    first=last=None;correct=[];ready=[];lengths=[];run=0
    chinese=[s for s in f['gold'][0]['segments'] if s['kind']=='chinese']
    first_end=chinese[0]['end'] if chinese else -1
    for s in episode:
        if f['events'][s['event_index']]['type']!='key':continue
        right=harm=eligible=False
        if s['query_id'] is not None:
            g=generators[s['query_id']];raw=g['raw'];eligible=f['intent']=='return' and first_end>=0 and len(raw)>=first_end
            if g['status']=='ok':
                if s['selected_id'] is not None:h=candidates[s['query_id']][s['selected_id']]['candidate']
                else:
                    segs=list(s['freeze'] or []);end=segs[-1]['end'] if segs else 0
                    if end<len(raw):segs.append({'kind':'literal','begin':end,'end':len(raw),'reading':''})
                    h={'display':s['display'],'display_segments':segs,'display_identity':identity([{'kind':'literal','begin':0,'end':len(raw),'reading':''}] if raw else [])}
                if f['intent']=='literal':right=h['display']==raw;harm=not right
                else:
                    import hashlib
                    right='h:'+hashlib.sha256(h['display_identity'].encode()).hexdigest() in projected_gold_ids(f,len(raw))
                    begin=chinese[0]['begin'];harm=any(x['kind']=='chinese' and x['begin']<min(begin,len(raw)) for x in h['display_segments'])
        index=len(correct);correct.append(right);ready.append(eligible)
        if eligible and right and first is None:first=index
        if harm:run+=1
        elif run:lengths.append(run);run=0
    stable=True
    for i in range(len(correct)-1,-1,-1):
        stable=stable and correct[i]
        if stable and (ready[i] or f['intent']=='literal'):last=i
    return first,last,lengths

def revision_counts(f,episode,candidates):
    previous=None;revisions=resolutions=0
    def projection(segments,n):
        result=[None]*n
        for s in segments:
            value=(s['kind'],s['begin'] if s['kind']=='chinese' else 0,s['end'] if s['kind']=='chinese' else 0,s['reading'])
            for i in range(s['begin'],min(n,s['end'])):result[i]=value
        return result
    for s in episode:
        if f['events'][s['event_index']]['type']!='key' or s['query_id'] is None:continue
        if s['selected_id'] is None:previous=None;continue
        h=candidates[s['query_id']][s['selected_id']]['candidate']
        if previous is not None:
            a=previous['display_segments'];b=h['display_segments'];n=min(a[-1]['end'] if a else 0,b[-1]['end'] if b else 0)
            revision=resolution=False
            for x,y in zip(projection(a,n),projection(b,n)):
                if x[0]!='chinese' and y[0]!='chinese':continue
                if x[0]=='chinese' and x[2]>n or y[0]=='chinese' and y[2]>n:continue
                if x[0]=='pending' and y[0]=='chinese':resolution=True
                elif x!=y:revision=True
            revisions+=int(revision);resolutions+=int(resolution)
        previous=h
    return revisions,resolutions

def validate_and_grade(fixtures,rows,snapshots,occurrences,generators,models):
    fs={f['id']:f for f in fixtures};gs={};previous={}
    for g in generators:
        q=g['query_id']
        if q in gs or q!=query_id(g['fixture_id'],g['condition'],g['event_index'],g['raw']):raise ValueError('duplicate/incorrect generator query')
        if g['kept_count']!=len(g['kept_ids']) or g['kept_count']>256:raise ValueError('kept count differs')
        counters=[g[k] for k in ('generate_calls','probe_lex_queries','best_calls')]
        if any(x is None for x in counters):raise Insufficient('generator counters unavailable')
        key=(g['fixture_id'],g['condition'])
        if any(a<b for a,b in zip(counters,previous.get(key,[0,0,0]))):raise ValueError('generator counters decreased')
        previous[key]=counters
        if counters[1]>250000 or counters[2]>100000:raise ValueError('generator exceeds locked budgets')
        if g['status']=='ok' and (g['uncapped_count']!=len(g['uncapped_ids']) or g['kept_ids']!=g['uncapped_ids'][:256] or g['cap_pruned']!=(len(g['uncapped_ids'])>256)):raise ValueError('cap evidence differs')
        gs[q]=g
    cs={}
    for o in occurrences:
        g=gs.get(o['query_id'])
        if g is None:raise ValueError('occurrence missing generator')
        if any(o[k]!=g[k] for k in ('fixture_id','component_id','condition','event_index','key_offset','raw')):raise ValueError('occurrence provenance differs')
        h=o['candidate'];query=cs.setdefault(o['query_id'],{})
        if h['id'] in query or h['id']!=hid(h['segments']) or h['identity']!=identity(h['segments']) or h['display_identity']!=identity(h['display_segments']):raise ValueError('candidate identity differs/duplicate')
        if o['allowed']!=(o['exclusion_reason']=='none'):raise ValueError('exclusion label differs')
        if o['score'] is not None:
            model=models.get(o['condition'].split('-')[0])
            if model is None or not math.isclose(score(model,h['features']),o['score'],rel_tol=0,abs_tol=1e-12):raise ValueError('candidate score differs from frozen model')
        query[h['id']]=o
    for q,g in gs.items():
        if set(cs.get(q,{}))!=set(g['kept_ids']+[g['literal_id']]):raise Insufficient('capped candidate/literal stream incomplete')
    events={};seen_queries=set()
    for s in snapshots:
        key=(s['episode_id'],s['condition']);f=fs.get(s['episode_id'])
        if f is None or s['event_index']>=len(f['events']):raise ValueError('snapshot fixture/event differs')
        if s['view']!=s['app']+s['prelude']+s['display']:raise ValueError('snapshot ledger differs')
        prior=events.setdefault(key,[])
        if prior and (s['event_index']<=prior[-1]['event_index'] or not s['app'].startswith(prior[-1]['app'])):raise ValueError('event order/app rewrite')
        q=s['query_id']
        if q is not None:
            g=gs.get(q)
            if g is None:raise Insufficient('snapshot generator missing')
            if q in seen_queries:raise ValueError('duplicate query snapshot')
            seen_queries.add(q)
            if any(s[a]!=g[b] for a,b in (('episode_id','fixture_id'),('condition','condition'),('event_index','event_index'),('key_offset','key_offset'))):raise ValueError('snapshot provenance differs')
            if g['status']=='ok' and s['condition']!='shared' and f['intent']!='compat':
                model=models.get(s['condition'].split('-')[0])
                if model is None:raise Insufficient('selection model unavailable')
                literal=cs[q][g['literal_id']];literal_score=score(model,literal['candidate']['features'])
                expected=g['literal_id'] if literal['allowed'] else None;best=-math.inf
                ordered=sorted((o for o in cs[q].values() if o['candidate']['id']!=g['literal_id'] and o['allowed']),key=lambda o:(o['candidate']['cut'],o['candidate']['identity']))
                for candidate in ordered:
                    if candidate['score'] is None:raise Insufficient('executed candidate score missing')
                    if candidate['score']>best:
                        best=candidate['score']
                        if best-literal_score>model['margin']:expected=candidate['candidate']['id']
                if expected!=s['selected_id']:raise ValueError('selection differs from frozen score/margin/tie rule')
            if s['selected_id'] is not None:
                selected=cs[q].get(s['selected_id'])
                if selected is None or not selected['allowed'] or selected['candidate']['display']!=s['display']:raise ValueError('selected candidate/display differs')
        elif s['selected_id'] is not None:raise ValueError('queryless selected ID')
        prior.append(s)
    if set(gs)!=seen_queries:raise Insufficient('generator lacks runtime snapshot')
    grades={}
    for row in rows:
        key=(row['fixture_id'],row['condition'])
        if key in grades:raise ValueError('duplicate row')
        f=fs.get(row['fixture_id']);episode=events.get(key,[])
        if f is None or len(episode)!=len(f['events']):raise Insufficient('episode evidence incomplete')
        if f['intent']=='compat':grades[key]={'support':row['support']};continue
        final=[s for s in episode if f['events'][s['event_index']]['type']=='key'][-1]
        if final['query_id'] is None:raise Insufficient('final key lacks query')
        g=gs[final['query_id']]
        if g['component_id']!=row['component_id']:raise ValueError('row/generator component differs')
        result={'support':g['status']}
        if g['status']=='ok':
            if final['selected_id'] is not None:h=cs[final['query_id']][final['selected_id']]['candidate']
            else:
                segs=list(final['freeze'] or []);end=segs[-1]['end'] if segs else 0
                if end<len(g['raw']):segs.append({'kind':'literal','begin':end,'end':len(g['raw']),'reading':''})
                h={'display':final['display'],'display_segments':segs,'cut':len(g['raw'])}
            boundary,surface,english=grade(f,h)
            gold={hid(x['segments']) for x in f['gold']}
            survived=bool(gold.intersection(g['kept_ids'])) if f['intent']!='literal' else g['literal_id'] in gold
            result.update(boundary=boundary,surface=surface,english=english,survived=survived)
            for field,want in [('final_boundary_error',boundary),('final_surface_error',surface),('final_english_error',english),('gold_survived',survived)]:
                if row[field] is None:raise Insufficient('row grade unavailable')
                if row[field]!=want:raise ValueError('row differs from independent final snapshot/occurrence grade')
        elif row['support']!=g['status']:raise ValueError('row support differs')
        if f['events'][-1]['type']=='enter' and episode[-1]['view']!=final['view']:raise ValueError('Enter changed final view')
        revisions,resolutions=revision_counts(f,episode,cs)
        if (revisions,resolutions)!=(row['revision_count'],row['resolution_count']):raise ValueError('revision/resolution ownership evidence differs')
        first,last,lengths=temporal(f,episode,cs,gs)
        if (first,last,lengths)!=(row['first_correct'],row['last_stable'],row['temporary_error_lengths']):raise ValueError('temporal evidence differs')
        grades[key]=result
    return grades

def derived(errors,union):
    specs=[('interaction_I',None)]+[('flow_gain',m) for m in ('E0','EL','EW','EC')]+[('context_extra_revision_gain',None)]
    functions=[]
    for name,model in specs:
        def function(sample,name=name,model=model):
            ids=[i for i in sample if all(i in errors[c]['return_boundary'] for c in CONDITIONS)]
            if not ids:return None
            sums={c:sum(errors[c]['return_boundary'][i] for i in ids) for c in CONDITIONS}
            if name=='interaction_I':value=sums['EC-revise']-sums['EL-revise']-sums['EW-revise']+sums['E0-revise']
            elif name=='flow_gain':value=sums[model+'-fixed']-sums[model+'-revise']
            else:value=(sums['EC-fixed']-sums['EC-revise'])-(sums['E0-fixed']-sums['E0-revise'])
            return value/len(ids)
        functions.append(function)
    intervals=bootstrap(union,functions)
    return [{'name':name,'model':model,'cohort':'all_return','value':function(union),'bootstrap95':ci,'valid_bootstrap':valid}
            for (name,model),function,(ci,valid) in zip(specs,functions,intervals)]

def summarize(fixtures,sources,bank,assignment,rows,snapshots,occurrences,generators,models,lock,lock_hash,model_hashes,split='test'):
    grades=validate_and_grade(fixtures,rows,snapshots,occurrences,generators,models)
    fs={f['id']:f for f in fixtures};components={c['id']:c for c in assignment['components']}
    qualification={'test_return_components':0,'test_literal_components':0,'test_english_components':0,'contrast_components':0,'category_components':{},'source_complete':source_qualification(fixtures,sources,bank,lock),'surface_gold_complete':True}
    for c in assignment['components']:
        if c['split']=='test':
            if c['primary_return'] is not None:
                qualification['test_return_components']+=1;f=fs[c['primary_return']]
                begin=next(s['begin'] for s in f['gold'][0]['segments'] if s['kind']=='chinese')
                qualification['test_english_components']+=int(begin>0)
                qualification['category_components'][f['category']]=qualification['category_components'].get(f['category'],0)+1
                qualification['contrast_components']+=int(c['primary_return_metadata']['contrast_eligible'])
            qualification['test_literal_components']+=int(c['primary_literal'] is not None)
    union=[c['id'] for c in assignment['components'] if c['split']==split and (c['primary_return'] is not None or c['primary_literal'] is not None)]
    errors={c:{name:{} for name in METRICS} for c in CONDITIONS};row_by_key={};missing=False;missing_rows=False;relevant_rows=0;complete=True;unsafe=False;safety={k:0 for k in SAFETY}
    for row in rows:
        key=(row['fixture_id'],row['condition']);row_by_key[key]=row
        c=components[row['component_id']];f=fs[row['fixture_id']]
        intent=f['intent'];meta=c.get('primary_'+intent+'_metadata')
        primary=c.get('primary_'+intent)==f['id']
        if row['intent']!=intent or row['primary']!=primary or row['primary_metadata']!=meta:raise ValueError('selected-primary metadata differs')
        if meta and (row['primary_category']!=meta['category'] or row['context_class']!=meta['context_class'] or row['contrast_eligible']!=meta['contrast_eligible']):raise ValueError('primary metadata copy differs')
        if c['split']!=split:continue
        relevant_rows+=1
        if row['condition']=='shared':continue
        if row['condition'] not in errors:raise ValueError('unknown condition')
        g=grades[key]
        if g['support']!='ok':missing=True
        for k in SAFETY:
            if row[k] is None:safety[k]=None;complete=False
            else:
                unsafe=unsafe or row[k]!=0
                if safety[k] is not None:safety[k]+=row[k]
        complete=complete and row['safety_status']=='complete'
        if primary and g['support']=='ok':
            values=errors[row['condition']]
            if intent=='return':
                values['return_boundary'][c['id']]=g['boundary'];values['return_surface'][c['id']]=g['surface']
                begin=next(s['begin'] for s in f['gold'][0]['segments'] if s['kind']=='chinese')
                if begin>0:values['return_english'][c['id']]=g['english']
            elif intent=='literal':values['literal_only'][c['id']]=g['surface']
    for c in assignment['components']:
        if c['split']==split:
            for fixture in (c['primary_return'],c['primary_literal']):
                if fixture is not None and any((fixture,condition) not in row_by_key for condition in CONDITIONS):missing=True;missing_rows=True
    conditions={}
    for condition in CONDITIONS:
        conditions[condition]={}
        for name,values in errors[condition].items():
            relevant=[c for c in assignment['components'] if c['split']==split and c['primary_literal' if name=='literal_only' else 'primary_return'] is not None]
            if name=='return_english':relevant=[c for c in relevant if next(s['begin'] for s in fs[c['primary_return']]['gold'][0]['segments'] if s['kind']=='chinese')>0]
            unknown=len(relevant)-len(values)
            m=metric(list(values.values())+[True]*unknown)
            m.update(observed_errors=sum(values.values()),unavailable_components=unknown)
            if unknown:m['status']='unavailable';m['value']=m['wilson95']=None;missing=True
            conditions[condition][name]=m
    specs=[('E0-revise','EL-revise','all_return'),('E0-revise','EW-revise','all_return'),('EL-revise','EC-revise','all_return'),('EW-revise','EC-revise','all_return')]+[(m+'-fixed',m+'-revise','all_return') for m in ('E0','EL','EW','EC')]+[('E0-revise','EC-revise','context_contrast'),('EL-revise','EC-revise','context_contrast')]
    comparisons=[]
    for baseline,challenger,cohort in specs:
        base={i:v for i,v in errors[baseline]['return_boundary'].items() if cohort!='context_contrast' or components[i]['primary_return_metadata']['contrast_eligible']}
        new=errors[challenger]['return_boundary']
        if any(i not in new for i in base):missing=True
        base={i:v for i,v in base.items() if i in new};new={i:new[i] for i in base}
        comparison=paired(baseline,challenger,cohort,base,new,union)
        expected=sum(c['split']==split and c['primary_return'] is not None and (cohort!='context_contrast' or c['primary_return_metadata']['contrast_eligible']) for c in assignment['components'])
        if comparison['n']!=expected:
            comparison.update(n=expected,relative_error_reduction=None,absolute_error_reduction_ci95=None,valid_bootstrap=0);missing=True
        comparisons.append(comparison)
    if relevant_rows==0 or missing_rows:safety={k:None for k in SAFETY};complete=False
    report={'version':'shift-return-v2','kind':'report','demo':assignment['demo'],'outcome':'INCONCLUSIVE','primary_condition':'EC-revise','conditions':conditions,'paired':comparisons,'qualification':qualification,'source_lock_sha256':lock_hash,'models_sha256':model_hashes,'diagnostics_path':'diagnostics.json','safety':safety,'derived':derived(errors,union),'safety_status':'complete' if complete else 'incomplete','safety_reason':'' if complete else 'one or more required safety counters unavailable'}
    if relevant_rows==0:report['safety_status']='not_run';report['safety_reason']='no relevant replay rows supplied; safety counters unknown'
    if missing:
        for d in report['derived']:d.update(value=None,bootstrap95=None,valid_bootstrap=0)
    apply_gates(report,rows,assignment,missing,unsafe,complete,sample_qualification(fixtures,bank,assignment),fixtures)
    VALIDATOR.validate(report)
    for metrics in report['conditions'].values():
        for m in metrics.values():validate_metric(m)
    return report

def sample_qualification(fixtures,bank,assignment):
    returns=('typo','identifier','digit_initial','uppercase','unknown','known')
    literals=('ordinary_english','account','phonetic_account','model_version','numeric_date_time','url_path','identifier','random_ascii')
    counts={};variants={};totals={}
    if len({f['prototype_id'] for f in fixtures if f['intent']=='compat'})>500:return False
    for t in bank:
        indices=variants.setdefault(t['prototype_id'],set())
        if t['variant_index'] in indices:return False
        indices.add(t['variant_index'])
        if t['variant_index']==0:
            totals[t['intent']]=totals.get(t['intent'],0)+1
            if totals[t['intent']]>3000:return False
            key=(t['intent'],t['category'],t['stratum']);counts[key]=counts.get(key,0)+1
    if any(counts.get(('return',c,s),0)!=125 for c in returns for s in ('1','2-4','5-16','17-64')):return False
    if any(counts.get(('literal',c,s),0)!=n for c in literals for s,n in zip(('1-8','9-32','33-128','129-512'),(94,94,94,93))):return False
    fs={f['id']:f for f in fixtures};splits={s:{'return':0,'literal':0} for s in ('train','dev','test','regression')};categories={};space=ret=0
    for c in assignment['components']:
        for intent in ('return','literal'):
            id=c['primary_'+intent]
            if id is None:continue
            splits[c['split']][intent]+=1
            if c['split']=='test':
                f=fs[id];key=(intent,f['category']);categories[key]=categories.get(key,0)+1
                if intent=='return':
                    ret+=1
                    space+=any(s['kind']=='chinese' and f['expected_raw'][s['end']-1]==' ' for s in f['gold'][0]['segments'])
    return all(splits[s][i]>=100 for s in ('train','dev') for i in ('return','literal')) and all(categories.get(('return',c),0)>=20 for c in returns) and all(categories.get(('literal',c),0)>=20 for c in literals) and ret>0 and space/ret>=.2

def apply_gates(report,rows,assignment,missing,unsafe,complete,samples_complete,fixtures):
    def unavailable(reason):return {'status':'unavailable','reason':reason}
    def test(passed,reason):return {'status':'pass' if passed else 'fail','reason':reason}
    q=report['qualification'];gates={}
    gates['G0']=unavailable('formal tool/data/cohort qualification or complete eight-condition evidence missing')
    if len(report['models_sha256'])==4 and samples_complete and q['source_complete'] and not report['demo'] and not missing and q['test_return_components']>=400 and q['test_literal_components']>=400 and q['test_english_components']>=400 and q['contrast_components']>=100:gates['G0']=test(True,'frozen qualification and complete evidence verified')
    ec=report['conditions']['EC-revise'];ret=ec['return_boundary']
    test_components={c['id'] for c in assignment['components'] if c['split']=='test'}
    recall=[r for r in rows if r['condition'] in ('EC-revise','shared') and r['intent']=='return' and r['primary'] and r['component_id'] in test_components]
    gates['G1']=test(sum(r['gold_survived'] is True for r in recall)/len(recall)>=.99,'capped full-gold recall, including unavailable episodes as misses') if recall else unavailable('candidate recall evidence unavailable')
    gates['G2']=test(ret['value']<=.1,'final boundary/reading errors; surface separately reported') if ret['value'] is not None else unavailable('return denominator unavailable')
    p0,p1=report['paired'][8:10]
    gates['G3']=unavailable('context improvement evidence insufficient (baseline zero, cohort/discordance, or bootstrap unavailable)')
    if all(p['relative_error_reduction'] is not None and p['discordant_components']>=20 and p['absolute_error_reduction_ci95'] is not None for p in (p0,p1)):
        gates['G3']=test(p0['relative_error_reduction']>=.3 and p0['absolute_error_reduction_ci95'][0]>0 and p1['absolute_error_reduction_ci95'][0]>0,'predeclared contrast EC against both E0 and EL')
    gates['G4']=unavailable('literal risk denominators unavailable')
    if ec['literal_only']['value'] is not None and ec['return_english']['value'] is not None:
        passed=all(ec[name]['value']<=.01 and ec[name]['wilson95'][1]<=.01 and all(ec[name]['errors']<=report['conditions'][base][name]['errors'] for base in ('E0-revise','EL-revise')) for name in ('literal_only','return_english'))
        gates['G4']=test(passed,'two component risk denominators and Wilson upper limits; author mandatory cases additionally required')
    mandatory={'kevin87','nina87','lisa0304','gpt4','h264','gpt4o','hello4','zoom3','hello2.4'};seen=set();passed=True;fs={f['id']:f for f in fixtures}
    for row in rows:
        f=fs[row['fixture_id']]
        if row['condition']=='EC-revise' and f['intent']=='literal' and f['expected_raw'] in mandatory:
            seen.add(f['expected_raw']);passed=passed and row['support']=='ok' and row['final_surface_error'] is False
    if not passed:gates['G4']=test(False,'specified author literal preservation case failed')
    elif seen!=mandatory and gates['G4']['status']=='pass':gates['G4']=unavailable('specified author preservation evidence missing')
    gates['G5']=test(not unsafe,'observed safety counters; source parity/regression additionally required') if complete and not missing else unavailable('required safety evidence incomplete')
    report['gates']=gates
    if all(g['status']=='pass' for g in gates.values()):report['outcome']='GO_OUTPUT_EXPERIMENT'
    elif all(g['status']!='unavailable' for g in gates.values()) and any(g['status']=='fail' for g in gates.values()):report['outcome']='NO_GO_METHOD'

def diagnostics(fixtures,rows,generators,occurrences,models,snapshots):
    categories={};variants={};fs={f['id']:f for f in fixtures};recall={}
    for row in rows:
        cat=categories.setdefault(row['condition'],{}).setdefault(row['primary_category'],{})
        variant=variants.setdefault(row['condition'],{})
        for name,field in [('boundary','final_boundary_error'),('surface','final_surface_error'),('english','final_english_error')]:
            metric_name='return_'+name if row['intent']=='return' else 'literal_only' if row['intent']=='literal' and name=='surface' else ''
            if not metric_name:continue
            if row['primary']:cat.setdefault(metric_name,[]).append(row[field])
            if row[field] is not None:
                key=row['component_id']+':'+metric_name;variant[key]=variant.get(key,False) or row[field]
    def category_metric(values):
        known=[v for v in values if v is not None];unknown=len(values)-len(known)
        m=metric(known+[True]*unknown);m.update(observed_errors=sum(known),unavailable_components=unknown)
        if unknown:m.update(status='unavailable',value=None,wilson95=None)
        return m
    categories={condition:{category:{name:category_metric(values) for name,values in names.items()} for category,names in cats.items()} for condition,cats in categories.items()}
    for g in generators:
        status='unavailable'
        if g['status']=='ok':
            gold=projected_gold_ids(fs[g['fixture_id']],len(g['raw']))
            kept=bool(gold.intersection(g['kept_ids']+[g['literal_id']]))
            uncapped=bool(gold.intersection(g['uncapped_ids']+[g['literal_id']]))
            status='survived' if kept else 'cap_loss' if uncapped else 'generator_miss'
        recall[g['query_id']]=status
    result={'categories':categories,'component_any_variant':variants,'recall_by_query':recall,'unavailable_primary_metrics':'full component n retained; unavailable rows charged as failures in errors, value/CI null; paired/derived CI unavailable'}
    lookup={}
    for o in occurrences:lookup.setdefault(o['query_id'],{})[o['candidate']['id']]=o
    previous={};changed_events=changed_spans=0;valid=True
    def glyphs(o):
        h=o['candidate'];tail=len(o['raw'])-h['tail_start'];end=len(h['display'])-tail
        if tail<0 or h['cut']>end:return None
        chars=h['display'][h['cut']:end]
        if len(chars)!=h['k']:return None
        spans=[s for s in h['display_segments'] if s['kind']=='chinese']
        return {(s['begin'],s['end'],s['reading']):c for s,c in zip(spans,chars)}
    for s in snapshots:
        if s['query_id'] is None:continue
        key=(s['episode_id'],s['condition'])
        if s['selected_id'] is None:previous.pop(key,None);continue
        current=lookup[s['query_id']][s['selected_id']]
        if key in previous:
            a,b=glyphs(previous[key]),glyphs(current)
            if a is None or b is None:valid=False;break
            changes=sum(k in b and b[k]!=v for k,v in a.items())
            if changes:changed_events+=1;changed_spans+=changes
        previous[key]=current
    result['homophone_surface_changes']=({'status':'complete','events':changed_events,'changed_spans':changed_spans,'reason':'same owned reading span changed its surface; ordinary appended spans excluded'} if valid else {'status':'unavailable','events':None,'changed_spans':None,'reason':'surface/readings cannot be aligned to Unicode codepoints'})
    if 'EC' in models:
        model=models['EC'];first={};last={};queries={}
        for o in sorted(occurrences,key=lambda o:(o['fixture_id'],o['event_index'],o['candidate']['cut'],o['candidate']['identity'])):
            if o['condition']!='EC-revise':continue
            h=o['candidate'];last[o['fixture_id']]=max(last.get(o['fixture_id'],0),o['event_index']);queries.setdefault(o['query_id'],[]).append(o)
            if h['readings']:first.setdefault((o['fixture_id'],h['cut'],h['first_reading']),h['features'][9:13])
        sensitivity=[]
        for query in queries.values():
            if query[0]['event_index']!=last[query[0]['fixture_id']]:continue
            literal=next(o['candidate'] for o in query if not o['candidate']['readings']);chosen=literal;literal_score=score(model,literal['features']);best=-math.inf
            for o in sorted(query,key=lambda o:(o['candidate']['cut'],o['candidate']['identity'])):
                h=o['candidate']
                if not o['allowed'] or not h['readings']:continue
                features=h['features'][:];features[9:13]=first[(o['fixture_id'],h['cut'],h['first_reading'])];value=score(model,features)
                if value>best:
                    best=value
                    if value-literal_score>model['margin']:chosen=h
            f=fs[query[0]['fixture_id']];boundary,surface,english=grade(f,chosen)
            sensitivity.append({'fixture_id':f['id'],'component_id':query[0]['component_id'],'selected_id':chosen['id'],'final_boundary_error':boundary,'final_surface_error':surface,'final_english_error':english})
        result['ec_first_W_sensitivity']={'coefficients':'same frozen EC','retrained':False,'main_arm':False,'rows':sorted(sensitivity,key=lambda x:x['fixture_id'])}
    return result

def selected_models(lock):
    paths=[path for path in lock['files'] if Path(path).name=='selected-model-manifest.json']
    if len(paths)!=1:raise ValueError('execution/evidence lock must contain exactly one selected-model-manifest.json')
    manifest=record(paths[0],'selected_model_manifest')
    if filehash(paths[0])!=lock['files'][paths[0]]:raise ValueError('selected manifest bytes changed')
    if manifest['demo'] and not lock['demo']:raise ValueError('formal selected manifest cannot be demo')
    frozen={str(Path(path).resolve()):sha for path,sha in lock['files'].items()}
    models={};hashes={}
    for name in ('E0','EL','EW','EC'):
        entry=manifest['models'][name];path=Path(entry['path'])
        if not path.is_absolute():raise ValueError('selected model paths must be absolute')
        if frozen.get(str(path.resolve()))!=entry['sha256'] or filehash(path)!=entry['sha256']:raise ValueError('selected model not frozen or bytes differ')
        model=record(path,'model')
        if model['name']!=name or model['demo']!=manifest['demo'] or model['code_lock_sha256']!=manifest['source_lock_sha256'] or model['fit_lock_sha256']!=manifest['fit_lock_sha256']:raise ValueError('selected model manifest provenance differs')
        models[name]=model;hashes[name]=entry['sha256']
    return models,hashes

def main():
    p=argparse.ArgumentParser()
    for name in ('fixtures','sources','bank','assignment','rows','events','candidates','generators','lock','out'):p.add_argument('--'+name,required=True)
    args=p.parse_args()
    try:
        lock=verify_lock(args.lock,[getattr(args,k) for k in ('fixtures','sources','bank','assignment','rows','events','candidates','generators')]+[__file__,Path(__file__).with_name('exchange.py'),Path(__file__).with_name('dictionary.py')])
        fixtures=list(records(args.fixtures,'fixture'));sources=record(args.sources,'sources');bank=list(records(args.bank,'bank_task'));assignment=record(args.assignment,'assignment')
        for f in fixtures:fixture_semantics(f)
        validate_assignment(fixtures,sources,assignment,args.fixtures)
        models,model_hashes=selected_models(lock)
        rows=list(records(args.rows,'row'));events=list(records(args.events,'snapshot'));candidates=list(records(args.candidates,'candidate_occurrence'));generators=list(records(args.generators,'generator_diagnostic'))
        result=summarize(fixtures,sources,bank,assignment,rows,events,candidates,generators,models,lock,filehash(args.lock),model_hashes)
        out=newout(args.out);write(out/'report.json',result)
        write(out/'diagnostics.json',diagnostics(fixtures,rows,generators,candidates,models,events))
        return 0 if result['outcome']=='GO_OUTPUT_EXPERIMENT' else 4 if result['outcome']=='NO_GO_METHOD' else 3
    except Insufficient as error:print(error,file=sys.stderr);return 3
    except (ValueError,OSError,KeyError,IndexError) as error:print(error,file=sys.stderr);return 2

if __name__=='__main__':sys.exit(main())
