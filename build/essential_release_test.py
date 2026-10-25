import unittest
import json
from unittest.mock import patch
from essential_release import (validate_changes, validate_compatibility_changes,
                               COMPATIBILITY_FILES, patch_plan, validate_run_receipts,
                               select_release)


class EssentialReleaseTests(unittest.TestCase):
    def plan(self, version='1.1.5'):
        return dict(tag='MatriX.145.Flow-v'+version, mode='api_compatibility',
                    candidate_commit='a'*40, baseline_commit='b'*40,
                    candidate_run=123, baseline_run=456)

    def test_future_patch_uses_the_same_selection_without_a_workflow_edit(self):
        for version in ('1.1.5', '1.1.6', '2.0.12'):
            plan = self.plan(version)
            repo = 'owner/Flow'
            receipts = [dict(head_sha=plan[key+'_commit'], conclusion='success',
                             status='completed', path=workflow,
                             repository=dict(full_name=repo))
                        for key, workflow in (('candidate', '.github/workflows/codeql.yml'),
                                              ('baseline', '.github/workflows/build.yml'))]
            with patch('essential_release.changed_since', return_value=[]), \
                    patch('essential_release.subprocess.check_output', side_effect=[json.dumps(x).encode() for x in receipts]):
                self.assertEqual(select_release(plan['tag'], repo, {'patch_validation':plan}),
                                 dict(validation_scope='essential', web_run_id='456'))

    def test_unscoped_release_defaults_full_without_contacting_github(self):
        with patch('essential_release.subprocess.check_output') as called:
            self.assertEqual(select_release('MatriX.145.Flow-v1.1.6', 'owner/Flow', {'patch_validation':self.plan()}),
                             dict(validation_scope='full', web_run_id=''))
            called.assert_not_called()

    def test_invalid_patch_metadata_fails_closed(self):
        for key, value in (('mode','anything'), ('candidate_run',True), ('baseline_run',0),
                           ('candidate_commit','--HEAD'), ('baseline_commit','x'*40)):
            plan = self.plan(); plan[key] = value
            with self.subTest(key=key), self.assertRaises(ValueError):
                patch_plan(plan['tag'], {'patch_validation':plan})
        plan = self.plan('1.1.0')
        with self.assertRaises(ValueError):
            patch_plan(plan['tag'], {'patch_validation':plan})

    def test_receipts_require_exact_source_workflow_repository_and_success(self):
        plan=self.plan(); repo='owner/Flow'
        candidate=dict(head_sha='a'*40, conclusion='success', status='completed',
                       path='.github/workflows/codeql.yml',repository=dict(full_name=repo))
        baseline=dict(head_sha='b'*40, conclusion='success', status='completed',
                      path='.github/workflows/build.yml',repository=dict(full_name=repo))
        validate_run_receipts(plan,repo,candidate,baseline)
        for key,value in (('head_sha','c'*40),('conclusion','failure'),('conclusion','cancelled'),
                          ('status','in_progress'),('path','.github/workflows/unrelated.yml'),
                          ('repository',dict(full_name='other/Flow'))):
            for index in range(2):
                runs=[dict(candidate),dict(baseline)]; runs[index][key]=value
                with self.subTest(key=key,index=index), self.assertRaises(ValueError):
                    validate_run_receipts(plan,repo,*runs)

    def test_reviewed_compatibility_patch_is_narrowly_eligible(self):
        validate_compatibility_changes(sorted(COMPATIBILITY_FILES) + ['RELEASE_1_1_5_NOTES.md'])
        for name in COMPATIBILITY_FILES:
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_changes([name])  # still forbidden after tested candidate

    def test_compatibility_scope_cannot_change_streaming_components(self):
        for name in ('server/torr/preload.go', 'server/torr/parser.go',
                     'server/torr/torrent.go', 'server/lt/lt.go',
                     'server/lt/lt_shim.cpp', 'server/torr/storage/torrstor/reader.go',
                     'server/web/api/arbitrary.go', 'server/go.mod', 'web/yarn.lock',
                     'build/_deps.sh', 'distribution/Update-Flow.ps1',
                     '../server/web/api/stream.go'):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_compatibility_changes([name])

    def test_ci_only_repair_and_empty_diff_are_eligible(self):
        validate_changes([])
        validate_changes(['.github/workflows/build.yml', 'build/ci-apt.sh',
                          'build/ci-integration.sh', 'build/workflow_policy_test.py',
                          'build/essential_release.py', 'build/essential_release_test.py',
                          'RELEASE_ACCEPTANCE.json', 'RELEASE_1_1_4_NOTES.md'])

    def test_runtime_and_build_input_changes_require_full_validation(self):
        for name in ('server/flow/session.go', 'server/go.mod', 'server/go.sum',
                     'web/yarn.lock', 'web/src/main.tsx', 'build/_common.sh',
                     'build/_deps.sh', 'build/linux-amd64.sh', 'build/release.py',
                     'build/native-patches/queue.patch', 'distribution/Update-Flow.ps1',
                     'Dockerfile.runtime', 'LICENSE', 'gen_web.go', 'web/README.md'):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_changes([name])

    def test_document_archive_does_not_allow_code_or_other_component_readmes(self):
        validate_changes(['docs/README.md', 'docs/history/RELEASE_1_1_6_NOTES.md',
                          'docs/security/SECURITY_DISPOSITIONS_PR10.md'])
        for name in ('docs/tool.py', 'docs/build.yml', 'docs/../server/README.md',
                     'web/README.md', 'docs\\history\\README.md'):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_changes([name])

    def test_path_tricks_fail_closed(self):
        for name in ('/README.md', '../README.md', '.github/workflows/../../server/a',
                     '.github\\workflows\\build.yml'):
            with self.subTest(name=name), self.assertRaises(ValueError):
                validate_changes([name])
