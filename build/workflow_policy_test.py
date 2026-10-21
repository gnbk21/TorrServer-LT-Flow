"""Reusable web jobs must not cancel different callers on the same PR."""
from pathlib import Path
import unittest

class WorkflowPolicyTests(unittest.TestCase):
    def test_reusable_web_concurrency_is_scoped_to_caller(self):
        text=(Path(__file__).resolve().parent.parent/'.github/workflows/build-web.yml').read_text()
        group=next(line for line in text.splitlines() if line.startswith('  group:'))
        self.assertIn('github.workflow',group)
        self.assertIn("github.event_name == 'pull_request'",text)

    def test_only_superseded_pr_builds_and_scans_are_cancelled(self):
        root=Path(__file__).resolve().parent.parent/'.github/workflows'
        for name in ['build.yml','codeql.yml']:
            text=(root/name).read_text()
            self.assertIn("cancel-in-progress: ${{ github.event_name == 'pull_request' }}",text)
