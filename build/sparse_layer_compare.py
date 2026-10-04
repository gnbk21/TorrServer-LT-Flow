#!/usr/bin/env python3
"""Repeated native-layer comparisons; only identical fixtures are compared."""
import argparse
import hashlib
import json
from pathlib import Path
import statistics

from fixtures import generate
from sparse_harness import run

CASES=('below','near','above','complementary','ten-suppliers','late-have')
METRICS=('buffer_gate_response_ms','urgent_first_request_ms_after_seek','first_peer_payload_ms','elapsed_ms')


def evaluate(binaries, fixtures, output):
    generate(fixtures,seconds=12)
    manifest=json.loads((fixtures/'fixtures.json').read_text(encoding='utf-8'))
    media=next(f for f in manifest['fixtures'] if f['name']=='variable.mp4')
    report={'http_delivery_only':True,'fixture':media,'variants':{},'runs':[],'passed':False}
    for label,path in binaries.items():
        report['variants'][label]={'executable_sha256':hashlib.sha256(path.read_bytes()).hexdigest()}
    output.mkdir(parents=True,exist_ok=True)
    try:
        for repetition in range(3):
            variants=list(binaries)
            # Rotate variant and fixture order, with fresh process/state per case.
            variants=variants[repetition:]+variants[:repetition]
            cases=list(CASES if repetition%2==0 else reversed(CASES))
            for label in variants:
                root=output/f'{repetition+1}-{label}'
                for case in cases:
                    print(f'Layer comparison {repetition+1} {label} {case}',flush=True)
                    result=run(binaries[label],root,fixtures/'variable.mp4',case,media['bytes']/media['duration'])
                    report['runs'].append({'variant':label,'repetition':repetition+1,**result})
                    (output/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
                    if not result['passed']:raise AssertionError(result.get('error','native case failed'))
        summary={}
        for label in binaries:
            summary[label]={}
            for case in CASES:
                matches=[r for r in report['runs'] if r['variant']==label and r['case']==case]
                values={}
                for metric in METRICS:
                    samples=[r[metric] for r in matches if r.get(metric) is not None]
                    if samples:values[metric]={'median':statistics.median(samples),'minimum':min(samples),'maximum':max(samples),'n':len(samples)}
                for i,name in enumerate(('head_ttfb_ms','seek_ttfb_ms','back_ttfb_ms')):
                    samples=[r['ranges'][i]['ttfb_ms'] for r in matches]
                    values[name]={'median':statistics.median(samples),'minimum':min(samples),'maximum':max(samples),'n':len(samples)}
                values['duplicate_bytes']=[r['flow'].get('sparse',{}).get('redundant_bytes') for r in matches]
                values['paced_client_read_wait_ms']=[r.get('paced_delivery',{}).get('client_read_wait_ms') for r in matches]
                summary[label][case]=values
        report['summary']=summary
        report['passed']=True
    except Exception as error:
        report['error']=str(error)
        raise
    finally:
        (output/'report.json').write_text(json.dumps(report,indent=2),encoding='utf-8')
    return report


if __name__=='__main__':
    parser=argparse.ArgumentParser()
    for label in ('original','stable','queue','candidate'):parser.add_argument('--'+label,type=Path,required=True)
    parser.add_argument('--fixtures',type=Path,required=True);parser.add_argument('--output',type=Path,required=True)
    args=parser.parse_args()
    evaluate({label:getattr(args,label).resolve() for label in ('original','stable','queue','candidate')},args.fixtures.resolve(),args.output.resolve())
