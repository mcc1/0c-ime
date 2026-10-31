#!/usr/bin/env python3
"""Final bounded public-source acquisition for mixed-dataset-v1.

Same ASCII original-token rule as USB; all direct device entries are scanned,
without policy/model queries or sample-count stopping. Subsystem rows are not
additional prototypes. Actual vendor names unite cross-catalog manufacturers.
"""
import hashlib
import json
import pathlib
import re
import shutil
import sys
from generate import OUT,row,literal,split_literal

COMMIT='19e7dfd4f141b895c36a50ba500ea96470065e27'
SHA='3af2de60a0cb598912d4ede5314fcb36a59593d08fba4f7f26ca4d4e53a7b9ea'

def main():
    pci=pathlib.Path(sys.argv[1]);usb=pathlib.Path(sys.argv[2]);b=(pci/'pci.ids').read_bytes();assert hashlib.sha256(b).hexdigest()==SHA
    rows=[json.loads(s) for s in (OUT/'fixtures.jsonl').read_text().splitlines()];assert not any(r['id'].startswith('pciids-') for r in rows)
    meta=json.loads((OUT/'metadata.json').read_text())
    vendor_names={}
    for prefix,source in [('usbids',usb/'usb.ids'),('pciids',pci/'pci.ids')]:
        for line in source.read_bytes().decode('utf-8',errors='replace').splitlines():
            if re.match(r'^[0-9a-f]{4}  ',line):
                vendor_names[(prefix,line[:4])]=line[6:].strip()
    for r in rows:
        if r['id'].startswith('usbids-'):
            vendor=meta['rows'][r['id']]['vendor'];r['families'].append('source:manufacturer:'+vendor_names['usbids',vendor])
    vendor='';count=0
    for lineno,line in enumerate(b.decode('latin1').splitlines(),1):
        if re.match(r'^[0-9a-f]{4}  ',line):vendor=line[:4]
        elif re.match(r'^\t[0-9a-f]{4}  ',line):
            device=line[1:5];label=line[7:]
            for ti,m in enumerate(re.finditer(r'(?<!\S)[A-Za-z][A-Za-z0-9._/-]{1,46}(?!\S)',label)):
                token=m.group()
                if not re.search('[0-9]',token):continue
                rid=f'pciids-{vendor}-{device}-{ti:02d}'
                ref=f'https://github.com/pciutils/pciids/blob/{COMMIT}/pci.ids#L{lineno}; vendor={vendor}; product={device}; exact ASCII whitespace-delimited token'
                r=row(rid,'literal',token,[[literal(w)] for w in split_literal(token)],['literal:'+token,'source:pciids-vendor-'+vendor,'template:pciids-product-family-'+vendor,'source:manufacturer:'+vendor_names['pciids',vendor]],ref,'GPL-2.0-or-later; redistributed under GPL-3.0 option; sources/pciids/NOTICE.txt and ../usbids/LICENSE')
                r['source']['kind']='public';rows.append(r);meta['rows'][rid]={'categories':['model','public-pciids'],'mandatory':False,'quality_eligible':True,'vendor':vendor,'product':device,'source_line':lineno};count+=1
    ids=[r['id'] for r in rows];assert len(set(ids))==len(ids)
    rows.sort(key=lambda r:r['id']);(OUT/'fixtures.jsonl').write_text(''.join(json.dumps(r,ensure_ascii=False,separators=(',',':'))+'\n' for r in rows));(OUT/'metadata.json').write_text(json.dumps(meta,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    d=OUT/'sources/pciids';d.mkdir(parents=True,exist_ok=True);shutil.copyfile(pci/'README.md',d/'README.md')
    (d/'NOTICE.txt').write_text('\n'.join(b.decode('latin1').splitlines()[:20])+'\n\nDerived original ASCII identifiers distributed under the GPL-3.0 option of GPL-2.0-or-later. Full GPLv3 text: ../usbids/LICENSE. Attribution: Martin Mares, Albert Pool and PCI ID Project contributors.\n')
    # Exact human-reviewed aliases for the same named manufacturer. Shared
    # factual tokens additionally create transitive auto-edges across vendors.
    names=set(vendor_names.values());aliases={}
    groups=[('Intel Corporation',['Intel Corp.']),('Microsoft Corporation',['Microsoft Corp.']),('NVIDIA Corporation',['NVidia Corp.','NVIDIA Corp.']),('Realtek Semiconductor Co., Ltd.',['Realtek Semiconductor Corp.']),('Advanced Micro Devices, Inc. [AMD/ATI]',['Advanced Micro Devices, Inc.','Advanced Micro Devices [AMD]']),('Broadcom Inc. and subsidiaries',['Broadcom Corp.']),('ASUSTek Computer Inc.',['ASUSTek Computer, Inc.']),('Samsung Electronics Co Ltd',['Samsung Electronics Co., Ltd'])]
    for canonical,variants in groups:
        if canonical not in names:continue
        for v in variants:
            if v in names and v!=canonical:aliases['source:manufacturer:'+v]='source:manufacturer:'+canonical
    (OUT/'aliases.json').write_text(json.dumps({'version':'mixed-family-v1','aliases':aliases},ensure_ascii=False,sort_keys=True,indent=2)+'\n')
    p=json.loads((OUT/'protocol.json').read_text());p['public_pciids']={'commit':COMMIT,'sha256':SHA,'source_rows':count,'rule':'all direct-device original ASCII tokens using the identical USB rule; vendor/manufacturer family grouping; no more sources after this acquisition','license':'GPL-2.0-or-later, GPL-3.0 redistribution option','attribution':'Martin Mares, Albert Pool and PCI ID Project contributors'};p['public_source_acquisition_closed']=True;(OUT/'protocol.json').write_text(json.dumps(p,ensure_ascii=False,indent=2)+'\n');print(json.dumps({'added_public_rows':count,'total_rows':len(rows),'manufacturer_aliases':len(aliases)}))

if __name__=='__main__':main()
