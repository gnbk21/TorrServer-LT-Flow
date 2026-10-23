import copy
import unittest

from recover_release import REPOSITORY, TAG, SOURCE, RUN, SECURITY_RUN, validate
from release import PLATFORMS, selected_platforms
import release_gates_test
from release_gates import validate_jobs


class RecoveryTests(unittest.TestCase):
    def evidence(self):
        run = dict(id=RUN, head_sha=SOURCE, head_branch=TAG, event='push', status='completed',
                   conclusion='failure', path='.github/workflows/ts_release.yml',
                   repository=dict(full_name=REPOSITORY))
        security = dict(run, id=SECURITY_RUN, path='.github/workflows/codeql.yml', conclusion='success')
        jobs = release_gates_test.ReleaseGateTests().rows()
        annotations = {}
        for job in jobs:
            if job['name'].startswith('macos / macos/'):
                job.update(conclusion='cancelled', runner_id=0, steps=[], labels=['macos-latest'])
                annotations[str(job['id'])] = [dict(message='The job was not acquired by Runner of type hosted even after multiple attempts'),
                                               dict(message='Due to capacity constraints, jobs targeting macOS arm64 runners may experience longer queue times.')]
        return run, security, jobs, annotations

    def test_explicit_subset_preserves_all_other_gates_and_default_requires_macos(self):
        evidence = self.evidence()
        self.assertEqual(validate(*evidence), 23)
        with self.assertRaises(ValueError): validate_jobs(evidence[2], 'essential')
        for job in evidence[2]:
            if job['name'].startswith('macos / macos/'): continue
            bad = copy.deepcopy(evidence)
            next(row for row in bad[2] if row['id']==job['id'])['conclusion']='skipped'
            with self.subTest(role=job['name']), self.assertRaises(ValueError): validate(*bad)

    def test_source_security_identity_and_infrastructure_failure_are_required(self):
        for index, key, value in ((0,'head_sha','b'*40), (0,'head_branch','other'),
                                  (0,'id',RUN+1), (0,'event','pull_request'),
                                  (1,'conclusion','failure'), (1,'status','in_progress')):
            bad = self.evidence(); bad[index][key]=value
            with self.subTest(key=key), self.assertRaises(ValueError): validate(*bad)
        for field, value in (('runner_id',123), ('steps',[dict(name='Build')]), ('conclusion','failure'), ('labels',['macos-15'])):
            bad = self.evidence()
            next(row for row in bad[2] if row['name']=='macos / macos/arm64')[field]=value
            with self.subTest(field=field), self.assertRaises(ValueError): validate(*bad)
        bad = self.evidence(); bad[3].clear()
        with self.assertRaises((ValueError, KeyError)): validate(*bad)

    def test_platform_scope_is_explicit_and_never_advertises_missing_macos(self):
        self.assertEqual(selected_platforms('all'), PLATFORMS)
        subset=selected_platforms('non-macos')
        self.assertEqual(len(subset),6)
        self.assertFalse(any(p.startswith('darwin-') for p in subset))
        with self.assertRaises(ValueError): selected_platforms('automatic')
