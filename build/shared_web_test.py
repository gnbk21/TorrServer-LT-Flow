import json
from pathlib import Path
import tempfile
import subprocess
import unittest
from shared_web import expected_identity, trusted_run, verify


class SharedWebTests(unittest.TestCase):
    def test_handoff_manifest_does_not_mark_native_builds_modified(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(['git', 'check-ignore', '-q', '--', 'server/web/pages/template/flow-web-source.json'], cwd=root, capture_output=True)
        self.assertEqual(result.returncode, 0, 'UI handoff metadata must not taint native VCS provenance')

    def test_source_and_trust_boundary(self):
        expected=expected_identity({'GITHUB_REPOSITORY':'owner/Flow','GITHUB_SHA':'a'*40,'FLOW_PR_HEAD':'b'*40,'FLOW_PR_NUMBER':'13'})
        run={'event':'pull_request','path':'.github/workflows/checks.yml','head_sha':'b'*40,'repository':{'full_name':'owner/Flow'},'pull_requests':[{'number':13,'head':{'sha':'b'*40}}]}
        self.assertTrue(trusted_run(run,expected))
        for key,value in [('event','pull_request_target'),('head_sha','c'*40),('repository',{'full_name':'attacker/Flow'}),('path','.github/workflows/other.yml')]:
            self.assertFalse(trusted_run(dict(run,**{key:value}),expected))
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory);(path/'flow-web-source.json').write_text(json.dumps(expected))
            verify(path,expected)
            with self.assertRaises(ValueError):verify(path,dict(expected,merge_sha='d'*40))
