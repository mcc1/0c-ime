#!/usr/bin/env python3
"""Append all fixed-rule original model tokens, before quality evaluation.

Argument is the directory containing pinned usb.ids, LICENSE, README.md. The
catalog is byte-oriented (not entirely valid UTF-8); ASCII tokens are extracted
without changing their spelling, using Latin-1 solely as a lossless byte map.
Vendor families conservatively unite every product of the same vendor.
"""
import hashlib
import json
import pathlib
import re
import shutil
import sys
from generate import OUT, row, literal, split_literal

COMMIT='5eb613762db1e34b9218da63a88f22dd26f743c5'
SHA='f5a48b0cc8dae1607c2f0bae6b8dc13f2ecef69dbeaeaf34b4be8e280d34dba4'

def main():
    src=pathlib.Path(sys.argv[1]);b=(src/'usb.ids').read_bytes()
    assert hashlib.sha256(b).hexdigest()==SHA
    license_out=OUT/'sources/usbids';license_out.mkdir(parents=True,exist_ok=True)
    for f in ['LICENSE','README.md']:
        shutil.copyfile(src/f,license_out/f)
    rows=[json.loads(s) for s in (OUT/'fixtures.jsonl').read_text().splitlines()]
    assert not any(r['id'].startswith('usbids-') for r in rows)
    meta=json.loads((OUT/'metadata.json').read_text())
    vendor='';vendor_name='';count=0
    for lineno,line in enumerate(b.decode('latin1').splitlines(),1):
        if re.match(r'^[0-9a-f]{4}  ',line):
            vendor,vendor_name=line[:4],line[6:]
        elif re.match(r'^\t[0-9a-f]{4}  ',line):
            device=line[1:5];label=line[7:]
            for ti,m in enumerate(re.finditer(r'(?<!\S)[A-Za-z][A-Za-z0-9._/-]{1,46}(?!\S)',label)):
                token=m.group()
                if not re.search('[0-9]',token):
                    continue
                rid=f'usbids-{vendor}-{device}-{ti:02d}'
                ref=f'https://github.com/usbids/usbids/blob/{COMMIT}/usb.ids#L{lineno}; vendor={vendor}; product={device}; exact ASCII whitespace-delimited token'
                r=row(rid,'literal',token,[[literal(w)] for w in split_literal(token)],['literal:'+token,'source:usbids-vendor-'+vendor,'template:usbids-product-family-'+vendor],ref,'GPL-3.0; LICENSE and attribution in sources/usbids/')
                r['source']['kind']='public';rows.append(r)
                meta['rows'][rid]={'categories':['model','public-usbids'],'mandatory':False,'quality_eligible':True,'vendor':vendor,'product':device,'source_line':lineno}
                count+=1
    ids=[r['id'] for r in rows];assert len(set(ids))==len(ids)
    rows.sort(key=lambda r:r['id'])
    (OUT/'fixtures.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False,separators=(',',':'))+'\n' for r in rows))
    (OUT/'metadata.json').write_text(json.dumps(meta,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    protocol=json.loads((OUT/'protocol.json').read_text())
    protocol['public_usbids']={'commit':COMMIT,'sha256':SHA,'source_rows':count,'rule':'all exact ASCII whitespace-delimited tokens matching [A-Za-z][A-Za-z0-9._/-]{1,46} and containing a digit; vendor-family grouping; Latin-1 byte map; no spelling transformation','license':'GPL-3.0','attribution':'Linux USB ID Repository, maintained by the USB ID database contributors; see upstream README/LICENSE'}
    (OUT/'protocol.json').write_text(json.dumps(protocol,ensure_ascii=False,indent=2)+'\n')
    print(json.dumps({'added_public_rows':count,'total_rows':len(rows)}))

if __name__=='__main__':main()
