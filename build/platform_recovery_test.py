import copy
import unittest
from platform_recovery import validate
from release import REPOSITORY
from release_gates_test import ReleaseGateTests

class PlatformRecoveryTests(unittest.TestCase):
    def evidence(self):
        run=dict(head_sha='a'*40,path='.github/workflows/ts_release.yml',status='completed',event='push',head_branch='MatriX.145.Flow-v2.0.0',repository=dict(full_name=REPOSITORY))
        security=dict(run,path='.github/workflows/codeql.yml',conclusion='success')
        jobs=ReleaseGateTests().rows()
        jobs += [dict(id=100+i,name=f'cross / high-bitrate-comparison ({i})',conclusion='success') for i in range(10)]
        jobs += [dict(id=200+i,name=f'cross / native-profiling / capture ({i})',conclusion='success') for i in range(2)]
        jobs += [dict(id=300+i,name=f'cross / Parallel streaming ({i})',conclusion='success') for i in range(6)]
        return run,security,jobs
    def test_full_source_identical_success_required(self):
        run,security,jobs=self.evidence();validate(run,security,jobs,run['head_branch'],'a'*40)
        for index in (0,1):
            bad=copy.deepcopy((run,security,jobs));bad[index]['head_sha']='b'*40
            with self.assertRaises(ValueError):validate(*bad,run['head_branch'],'a'*40)
        jobs[-1]['conclusion']='skipped'
        with self.assertRaises(ValueError):validate(run,security,jobs,run['head_branch'],'a'*40)
    def test_missing_platform_requires_owner_selection_and_never_hides_code_failure(self):
        run,security,jobs=self.evidence()
        row=next(row for row in jobs if row['name']=='macos / macos/arm64');row.update(conclusion='cancelled',runner_id=0,steps=[])
        with self.assertRaises(ValueError):validate(run,security,jobs,run['head_branch'],'a'*40)
        self.assertEqual(validate(run,security,jobs,run['head_branch'],'a'*40,True)[1],['darwin-arm64'])
        row['runner_id']=10
        with self.assertRaises(ValueError):validate(run,security,jobs,run['head_branch'],'a'*40,True)
