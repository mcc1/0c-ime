#!/usr/bin/env python3
"""Build a Windows x64 upgrade-test helper; requires a verified official 0.2.10 installation."""
import argparse, hashlib, json, os, shutil, subprocess, zipfile
from pathlib import Path
from collect_notices import go_sections
from package_windows import check_pe, pe_imports
ROOT = Path(__file__).resolve().parents[1]
LOADER_SHA = '86545b66cdb0603bc26b626fb9ad610cb6e71f28d468f5ea66df23b03dda96d5'
def sha(p): return hashlib.sha256(Path(p).read_bytes()).hexdigest()
def run(a, **kw): return subprocess.run(a, cwd=ROOT, check=True, **kw)
def build(a):
    if subprocess.check_output(['git','status','--porcelain'],cwd=ROOT).strip(): raise ValueError('Commit the build sources first.')
    revision=subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()
    out=a.out.resolve(); work=ROOT/'build'/('windows-feature-preview-'+a.version)
    if out.exists() or out.with_suffix('.zip').exists() or work.exists(): raise ValueError('Use fresh output/version paths.')
    if sha(a.loader)!=LOADER_SHA: raise ValueError('Unmodified Microsoft WebView2 loader hash differs.')
    work.mkdir(parents=True); env=dict(os.environ,GOOS='windows',GOARCH='amd64',CGO_ENABLED='0')
    for package in ('0c-server','0c-update'):
        run([a.go,'build','-trimpath','-buildvcs=false','-ldflags','-s -w -X main.buildVersion='+a.version,
             '-o',str(work/(package+'.exe')),'./cmd/'+package],env=env)
    for path,name in [(a.settings,'0c-settings.exe'),(a.loader,'WebView2Loader.dll'),(a.settings_notices,'SETTINGS-NOTICES.txt')]: shutil.copyfile(path,work/name)
    executables=('0c-server.exe','0c-update.exe','0c-settings.exe','WebView2Loader.dll')
    deps={}
    for name in executables:
        check_pe(work/name); deps[name]=pe_imports(work/name)
        if any(x.lower().startswith(('libgcc_','libstdc++','libwinpthread','vcruntime','msvcp')) for x in deps[name]): raise ValueError('External C/C++ runtime dependency: '+name)
    if a.version.encode() not in (work/'0c-settings.exe').read_bytes(): raise ValueError('Settings version differs.')
    tokens={'VERSION':a.version,'SOURCE_REVISION':revision,'ENGINE_SHA':sha(work/'0c-server.exe'),'UPDATER_SHA':sha(work/'0c-update.exe'),
            'SETTINGS_SHA':sha(work/'0c-settings.exe'),'LOADER_SHA':sha(work/'WebView2Loader.dll'),'NOTICES_SHA':sha(work/'SETTINGS-NOTICES.txt')}
    text=(ROOT/'packaging/windows-preview/main.go.in').read_text()
    for key,value in tokens.items(): text=text.replace('@'+key+'@',value)
    (work/'main.go').write_text(text); shutil.copyfile(ROOT/'packaging/windows-preview/main_test.go.in',work/'main_test.go')
    relative='./'+work.relative_to(ROOT).as_posix()
    for command in ([a.go,'test',relative],[a.go,'vet',relative],[a.go,'run',relative,'--verify-payload']): run(command)
    out.mkdir(parents=True); program=out/'0c-windows-mixed-preview-setup.exe'
    run([a.go,'build','-trimpath','-buildvcs=false','-ldflags','-s -w','-o',str(program),relative],env=env); check_pe(program)
    source=out/'source'; source.mkdir()
    for name in ('main.go','main_test.go'): shutil.copyfile(work/name,source/name)
    with (source/('0c-ime-'+revision[:12]+'.tar.gz')).open('wb') as f: run(['git','archive','--format=tar.gz','--prefix=0c-ime/',revision],stdout=f)
    shutil.copyfile(ROOT/'LICENSE',out/'LICENSE'); shutil.copyfile(a.settings_notices,out/'SETTINGS-NOTICES.txt')
    notices='\n'.join(go_sections(a.go,'windows',['./cmd/0c-server','./cmd/0c-update',relative]))
    (out/'THIRD_PARTY_NOTICES.txt').write_text(notices+'\n'+a.settings_notices.read_text())
    for name,arg in {'Aggressive':'--policy aggressive','Conservative':'--policy conservative','Neighbor-On':'--neighbor on','Neighbor-Off':'--neighbor off',
                     'Manual-Updates':'--updates manual','Auto-Updates':'--updates auto','Rollback':'--rollback'}.items():
        (out/(name+'.cmd')).write_bytes(('@echo off\r\n"%~dp0'+program.name+'" '+arg+'\r\n').encode())
    text=(ROOT/'packaging/windows-preview/README.txt.in').read_text()
    for key,value in tokens.items(): text=text.replace('@'+key+'@',value)
    (out/'README.txt').write_text(text,encoding='utf-8-sig')
    metadata={'version':a.version,'revision':revision,'required_base':'official Windows amd64 0.2.10','payloads':tokens,'PE_imports':deps,
              'native_Windows_execution':'not_run','settings_target':'x86_64-pc-windows-gnu','rust':'1.99.0','installer_sha256':sha(program)}
    (out/'BUILD-INFO.json').write_text(json.dumps(metadata,ensure_ascii=False,indent=2)+'\n')
    paths=sorted(p for p in out.rglob('*') if p.is_file())
    (out/'SHA256SUMS').write_text(''.join(sha(p)+'  '+p.relative_to(out).as_posix()+'\n' for p in paths))
    archive=out.with_suffix('.zip')
    with zipfile.ZipFile(archive,'x',zipfile.ZIP_DEFLATED,compresslevel=9) as z:
        for p in sorted(out.rglob('*')):
            if p.is_file(): z.write(p,p.relative_to(out).as_posix())
    with zipfile.ZipFile(archive) as z:
        assert z.testzip() is None
        for p in sorted(out.rglob('*')):
            if p.is_file(): assert hashlib.sha256(z.read(p.relative_to(out).as_posix())).hexdigest()==sha(p)
    archive.with_suffix('.zip.sha256').write_text(sha(archive)+'  '+archive.name+'\n')
    print(json.dumps({'archive':str(archive),'sha256':sha(archive),'revision':revision},indent=2))
if __name__=='__main__':
    p=argparse.ArgumentParser(description=__doc__); p.add_argument('--version',default='0.2.11-preview.mixed.1')
    for name in ('settings','settings-notices','loader','out'): p.add_argument('--'+name,required=True,type=Path)
    p.add_argument('--go',default='go'); build(p.parse_args())
