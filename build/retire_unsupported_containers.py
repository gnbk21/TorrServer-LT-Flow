"""One-time owner-approved retirement; preserve the complete 1.1.7 manifest graph."""
import hashlib
import json
import os
from pathlib import Path
from urllib.request import Request, urlopen

REPO = 'gnbk21/TorrServer-LT-Flow'
IMAGE = 'gnbk21/torrserver-lt-flow'
TAG = 'MatriX.145.Flow-v1.1.7'
CURRENT = 'sha256:48f0a0e9a1f14b7200f910fc77025ce66691bb87aa014e28057cb42ceaf3ec0f'
OLD_TAGS = {'MatriX.145.Flow-v'+v for v in (
    '1.1.6', '1.1.5', '1.1.4', '1.1.3', '1.0.1', '0.2.0-preview.7',
    '0.2.0-preview.5', '0.2.0-preview.4', '0.2.0-preview.2')}
OLD_TAGS.add('MatriX.145.Flow-preview.2')
ACCEPT = ', '.join(('application/vnd.oci.image.index.v1+json',
    'application/vnd.docker.distribution.manifest.list.v2+json',
    'application/vnd.oci.image.manifest.v1+json',
    'application/vnd.docker.distribution.manifest.v2+json'))
OUTPUT = Path('retirement-evidence.json')


def github(path, method='GET'):
    request = Request('https://api.github.com/'+path, method=method, headers={
        'Authorization':'Bearer '+os.environ['GH_TOKEN'],
        'Accept':'application/vnd.github+json', 'X-GitHub-Api-Version':'2022-11-28'})
    with urlopen(request, timeout=30) as response:
        data=response.read()
        return json.loads(data) if data else None


def versions():
    rows=[]
    for page in range(1, 100):
        batch=github(f'users/gnbk21/packages/container/torrserver-lt-flow/versions?per_page=100&page={page}')
        rows.extend(batch)
        if len(batch)<100:return rows
    raise RuntimeError('Unexpected package catalog size')


def main():
    if os.environ.get('GITHUB_REPOSITORY') != REPO or os.environ.get('RETIRE_CONFIRM') != 'withdraw-before-1.1.7':
        raise RuntimeError('Explicit owner-approved scope required')
    latest=github('repos/'+REPO+'/releases/latest')
    assert latest['tag_name']==TAG and not latest['draft'] and len(latest['assets'])==35
    with urlopen('https://ghcr.io/token?service=ghcr.io&scope=repository:'+IMAGE+':pull',timeout=20) as response:
        token=json.load(response)['token']
    def manifest(ref):
        request=Request('https://ghcr.io/v2/'+IMAGE+'/manifests/'+ref,headers={
            'Authorization':'Bearer '+token,'Accept':ACCEPT})
        with urlopen(request,timeout=30) as response:
            data=response.read();digest='sha256:'+hashlib.sha256(data).hexdigest()
            assert digest==response.headers['Docker-Content-Digest']
            if ref.startswith('sha256:'):assert ref==digest
            return json.loads(data),digest
    def graph(ref):
        found=set(); pending=[ref]
        while pending:
            value,digest=manifest(pending.pop())
            if digest in found:continue
            found.add(digest)
            pending.extend(row['digest'] for row in value.get('manifests',[]))
        return found
    assert manifest(TAG)[1]==CURRENT and manifest('latest')[1]==CURRENT
    keep=graph(CURRENT)
    assert len(keep)==7, 'Unexpected 1.1.7 manifest graph'
    catalog=versions(); by_digest={row['name']:row for row in catalog}
    assert keep<=by_digest.keys()
    roots=[]; obsolete=set()
    for row in catalog:
        tags=set(row['metadata']['container']['tags'])
        if not tags & OLD_TAGS:continue
        assert tags <= OLD_TAGS|{'preview'}, 'Unreviewed alias on retired image'
        roots.append(row['name']);obsolete.update(graph(row['name']))
    assert len(roots)==10, 'Unexpected old release root count'
    obsolete-=keep
    assert len(obsolete)==70 and obsolete<=by_digest.keys(), 'Unreviewed retirement catalog'
    for digest in obsolete:
        tags=set(by_digest[digest]['metadata']['container']['tags'])
        assert tags<=OLD_TAGS|{'preview'}
    receipt=dict(repository=REPO, kept_tag=TAG, kept_index=CURRENT,
        protected_manifests=sorted(keep), requested=[dict(id=by_digest[d]['id'],digest=d,
            tags=by_digest[d]['metadata']['container']['tags']) for d in sorted(obsolete)],deleted=[],completed=False)
    def save():OUTPUT.write_text(json.dumps(receipt,indent=2)+'\n',encoding='utf-8')
    save()
    if os.environ.get('RETIRE_EXECUTE') != 'true':
        print(f'Preview only: {len(obsolete)} old manifests selected; {len(keep)} current manifests protected.')
        return
    # Remove obsolete indexes first. Shared/current manifests are never selected.
    ordered=sorted(obsolete,key=lambda d:(d not in roots,d))
    for digest in ordered:
        row=by_digest[digest]
        github(f"users/gnbk21/packages/container/torrserver-lt-flow/versions/{row['id']}",method='DELETE')
        receipt['deleted'].append(row['id']);save()
    remaining={row['name'] for row in versions()}
    assert not remaining&obsolete and keep<=remaining
    assert manifest(TAG)[1]==CURRENT and manifest('latest')[1]==CURRENT
    assert graph(CURRENT)==keep
    receipt.update(completed=True,remaining_versions=len(remaining));save()
    print(f'Retired {len(obsolete)} old manifests; all {len(keep)} current manifests preserved.')


if __name__=='__main__':main()
