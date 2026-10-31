#!/usr/bin/env python3
"""Generate the fixed mixed-dataset-v1 without querying any policy/model.

Run from repository root. Dataset is synthetic; literal templates are deliberately
unioned, not claimed independent. Public lexical facts retain source attribution.
"""
import hashlib
import json
import pathlib
import string

ROOT = pathlib.Path(__file__).resolve().parents[2]
OUT = ROOT/'testdata/mixed-boundary'
SEED = '20261101'
PAIRS = ['1ㄅ','qㄆ','aㄇ','zㄈ','2ㄉ','wㄊ','sㄋ','xㄌ','eㄍ','dㄎ','cㄏ','rㄐ','fㄑ','vㄒ','5ㄓ','tㄔ','gㄕ','bㄖ','yㄗ','hㄘ','nㄙ','uㄧ','jㄨ','mㄩ','8ㄚ','iㄛ','kㄜ',',ㄝ','9ㄞ','oㄟ','lㄠ','.ㄡ','0ㄢ','pㄣ',';ㄤ','/ㄥ','-ㄦ','6ˊ','3ˇ','4ˋ','7˙']
KEYS = {x[1]:x[0] for x in PAIRS}

def sha(s):
    return hashlib.sha256(s.encode()).hexdigest()

def spell(reading):
    return ''.join(KEYS[r] for r in reading)

def events(raw):
    return [{'type':'key','key':k} for k in raw]+[{'type':'finalize','reason':'enter'}]

def literal(raw):
    return [{'kind':'literal','begin':0,'end':len(raw),'text':raw}]

def partition(prefix, readings):
    p = []
    at = len(prefix)
    if prefix:
        p.append({'kind':'literal','begin':0,'end':at,'text':prefix})
    for r in readings:
        end = at+len(spell(r))
        p.append({'kind':'chinese','begin':at,'end':end,'reading':r})
        at = end
    return p

def split_literal(raw):
    windows = []
    start = 0
    tones = 0
    for i,k in enumerate(raw):
        if i-start == 64 or tones == 2:
            windows.append(raw[start:i])
            start, tones = i,0
        tones += k in '3467'
    windows.append(raw[start:])
    return windows

def row(rid,intent,raw,parts,families,reference,license='AGPL-3.0-only',prelude=None,text=None):
    # parts is a sequence of window partitions; final assertions occur at each
    # window's last consumed byte, including literal windows that have no tone.
    ws = split_literal(raw)
    assert len(ws)==len(parts),(rid,ws,parts)
    ats=[];n=0
    for w in ws:
        n+=len(w);ats.append(n-1)
    assertions=[]
    for wi,(at,ps) in enumerate(zip(ats,parts)):
        assertions.append({'at':at,'window':wi,'policy':'generator','stage':'final','accepted':ps})
    if text:
        assertions[-1]['accepted_text']=[text]
    return {'version':'mixed-fixture-v1','id':rid,'intent':intent,'source':{'kind':'synthetic','reference':reference,'license':license},'families':families,'prelude':prelude or [],'custom_tokens':[],'events':events(raw),'assertions':assertions}


def main():
    if (OUT/'fixtures.jsonl').exists():
        raise SystemExit('refusing to overwrite frozen fixtures')
    proto=json.loads((OUT/'protocol.json').read_text())
    words=[]
    for line in (ROOT/'data/lexicon/words.tsv').read_text().splitlines():
        if not line or line.startswith('#'):
            continue
        cols=line.split('\t');r=cols[1].split()
        if len(r)!=2 or any(x[-1] not in 'ˊˇˋ˙' or any(c not in KEYS for c in x) for x in r):
            continue
        if any(c in cols[0] for c in '\t\r\n'):
            continue
        words.append((sha(SEED+'\n'+cols[0]+'\n'+cols[1]),cols[0],r,cols[4]))
    words.sort()
    unique=[];seen=set()
    for _,w,r,origin in words:
        if w not in seen:
            seen.add(w);unique.append((w,r,origin))
    assert len(unique)>=proto['budgets']['lexical_chinese']
    english=[];english_all=set()
    for line in (ROOT/'data/english/esdb.tsv').read_text().splitlines():
        if line.startswith('#'):
            continue
        w=line.split('\t')[0];english_all.add(w)
        if 4<=len(w)<=12 and all(c in string.ascii_lowercase for c in w):
            english.append(w)
    english=sorted(set(english),key=lambda w:sha(SEED+'\nenglish\n'+w))
    rows=[];meta={}
    def add(r,categories,mandatory=False):
        rows.append(r);meta[r['id']]={'categories':categories,'mandatory':mandatory,'quality_eligible':any(a['policy']=='generator' and a['stage']=='final' for a in r['assertions'])}
    for i,(w,r,origin) in enumerate(unique[:proto['budgets']['lexical_chinese']]):
        cat=proto['chinese_templates'][i%8]
        prefix=english[i]
        prelude=[]
        if cat=='unknown-prefix':
            prefix='QZX'+prefix
        elif cat=='capitalized-prefix':
            prefix=prefix.upper()
        elif cat=='digit-initial':
            # Category membership only when the actual first body is digit-initial.
            if spell(r[0])[0] not in '125':
                cat='known-prefix'
        elif cat=='english-tail-collision':
            prefix+='s'
            if prefix not in english_all: cat='known-prefix'
        elif cat=='invalid-first-resumption':
            # First failed tone consumes an opportunity. Correct Chinese begins
            # in the following window after its second opportunity, as declared.
            prefix='QQQ3'+prefix
        elif cat=='direct':
            prefix=''
        elif cat=='context':
            prelude=[{'type':'key','key':k} for k in 'ru,3']
        raw=prefix+''.join(map(spell,r))
        ws=split_literal(raw)
        if len(ws)==1:
            parts=[[partition(prefix,r)]]
        else:
            # Preserve the final correct Chinese reading in its actual window;
            # first window includes the failed tone and completed first syllable.
            cut=len(ws[0]);firstend=len(prefix)+len(spell(r[0]))
            assert cut==firstend,(raw,ws)
            parts=[[partition(prefix,[r[0]])],[partition('',[r[1]])]]
        ref=f'dictionary-guided synthetic prototype; data/lexicon/words.tsv; word={w}; origin={origin}; attribution/licenses=data/sources/ledger.tsv and data/sources/licenses/'
        fam=[f'word:{w}',f'source:lexical-prototype-{w}']
        if prefix:
            fam.append('prefix:'+prefix)
        if prelude:
            fam.append('context:解')
        add(row(f'chinese-{i:04d}','chinese',raw,parts,fam,ref,'See data/sources/ledger.tsv per origin',prelude),[cat])
    for i in range(proto['budgets']['synthetic_literal']):
        cat=proto['literal_templates'][i%8];w=english[i+3000]
        if cat=='account': raw=w+'87'
        elif cat=='model': raw=w.upper()+'264'
        elif cat=='version': raw=w+'2.4.6'
        elif cat=='date': raw=f'{2000+i%27:04d}-{1+(i//27)%12:02d}-{1+(i//324)%28:02d}'
        elif cat=='decimal': raw=str(1000+i)+'.'+str((i*17)%1000).zfill(3)
        elif cat=='time': raw=f'{i%24:02d}:{(i//24)%60:02d}:{(i//1440)%60:02d}'
        elif cat=='url': raw='https://'+w+'.example/v2'
        else: raw=w+'_v4'
        ws=split_literal(raw)
        add(row(f'literal-{i:04d}','literal',raw,[[literal(v)] for v in ws],['literal:'+raw,'template:literal-'+cat],f'synthetic {cat} prototype; mixed-dataset-v1; no observed user input'),[cat])
    for i,(w,r,origin) in enumerate(unique[3000:3000+proto['budgets']['ambiguous']]):
        prefix='AMB'+english[6000+i];raw=prefix+''.join(map(spell,r))
        add(row(f'ambiguous-{i:04d}','ambiguous',raw,[[literal(raw),partition(prefix,r)]],['word:'+w,'prefix:'+prefix,'literal:'+raw,'source:lexical-prototype-'+w],'explicit synthetic dual-intent alternatives; no top-one label'),['ambiguous'])
    mandatory=[('593','', ['ㄓㄞˇ'],'窄'),('microsoft593fmp6','microsoft',['ㄓㄞˇ','ㄑㄩㄣˊ'],'microsoft窄群'),('WQ5j04fu06','WQ',['ㄓㄨㄢˋ','ㄑㄧㄢˊ'],'WQ賺錢'),('hello5j04fu06','hello',['ㄓㄨㄢˋ','ㄑㄧㄢˊ'],'hello賺錢'),('hellosu3cl3','hello',['ㄋㄧˇ','ㄏㄠˇ'],'hello你好'),('testsu.3','tests',['ㄧㄡˇ'],'tests有'),('appsul4','apps',['ㄧㄠˋ'],'apps要'),('useru/4','user',['ㄧㄥˋ'],'user應'),('api2k7','api',['ㄉㄜ˙'],'api的')]
    for i,(raw,p,r,txt) in enumerate(mandatory):
        add(row(f'mandatory-chinese-{i:02d}','chinese',raw,[[partition(p,r)]],['prefix:'+p if p else 'template:direct-mandatory','word:'+txt[len(p):],'source:mandatory-regression'], 'hand-transcribed source tests and evaluator contract',text=txt),['mandatory','digit-initial' if raw=='593' else 'known-prefix'],True)
    r=row('mandatory-context','chinese','WQ5j04fu06',[[partition('WQ',['ㄓㄨㄢˋ','ㄑㄧㄢˊ'])]],['prefix:WQ','word:賺錢','context:解','source:mandatory-regression'],'fixture-declared prelude/run boundary',prelude=[{'type':'key','key':k} for k in 'ru,3'])
    add(r,['mandatory','context'],True)
    for i,raw in enumerate(['kevin87','nina87','lisa0304','gpt4','h264','gpt4o']):
        add(row(f'mandatory-literal-{i:02d}','literal',raw,[[literal(v)] for v in split_literal(raw)],['literal:'+raw,'source:mandatory-regression'],'manual literal intent from acceptance protocol'),['mandatory','account' if i<3 else 'model'],True)
    # Contract state examples are tests, not training labels or measured success.
    for r in map(json.loads,(ROOT/'design/mixed-input-experiment/examples.jsonl').read_text().splitlines()):
        if r['id'] in ('selection-original','overflow-first-tone'):
            r['id']='state-'+r['id'];add(r,['state','original' if 'selection' in r['id'] else 'overflow'],True)
    rows.sort(key=lambda r:r['id'])
    with (OUT/'fixtures.jsonl').open('x',encoding='utf-8') as f:
        for r in rows:
            f.write(json.dumps(r,ensure_ascii=False,separators=(',',':'))+'\n')
    with (OUT/'metadata.json').open('x',encoding='utf-8') as f:
        f.write(json.dumps({'version':'mixed-metadata-v1','rows':meta},ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    (OUT/'aliases.json').write_text('{"version":"mixed-family-v1","aliases":{}}\n')
    print(json.dumps({'rows':len(rows),'intents':{k:sum(r['intent']==k for r in rows) for k in ['chinese','literal','ambiguous']}}))

if __name__=='__main__':
    main()
