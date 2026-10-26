"""Reusable web jobs must not cancel different callers on the same PR."""
from pathlib import Path
import re
import unittest

ROOT = Path(__file__).resolve().parent.parent


def job(workflow, name):
    """Read a top-level job without adding a YAML dependency to portable tests."""
    text = (ROOT / '.github/workflows' / workflow).read_text()
    match = re.search(r'^  ' + re.escape(name) + r':\n(.*?)(?=^  [\w-]+:|\Z)',
                      text, re.M | re.S)
    assert match, name
    return match[1]


class WorkflowPolicyTests(unittest.TestCase):
    def test_staged_release_keeps_gates_and_defers_public_container(self):
        classify = job('ts_release.yml', 'classify')
        self.assertIn('stage_before_publish', classify)
        self.assertIn('==os.environ["FLOW_TAG"]', classify)
        release = job('ts_release.yml', 'release')
        self.assertIn("draft: ${{ needs.classify.outputs.staged == 'true' }}", release)
        self.assertIn("needs.classify.outputs.staged != 'true' && needs.classify.outputs.channel == 'stable'", release)
        self.assertIn('needs: [classify, cross, macos]', release)
        self.assertIn("if: needs.classify.outputs.staged != 'true'", job('ts_release.yml', 'docker'))

    def test_resumed_container_requires_published_exact_successful_tagged_source(self):
        text = job('build-docker.yml', 'docker')
        gate = text.index('Validate published source')
        self.assertLess(gate, text.index('Download linux binaries'))
        self.assertLess(gate, text.index('Login to GitHub Container Registry'))
        for guard in ('r["status"]=="completed"', 'r["conclusion"]=="success"',
                      'r["head_sha"]==os.environ["FLOW_SOURCE"]',
                      'r["path"]==".github/workflows/ts_release.yml"',
                      'not p["draft"]', 'git rev-parse "$FLOW_VERSION^{commit}"'):
            self.assertIn(guard, text)

    def test_active_comparisons_do_not_depend_on_withdrawn_releases(self):
        for workflow, name in [('build.yml', 'high-bitrate-comparison'),
                               ('upgrade-benchmarks.yml', 'compare')]:
            text = job(workflow, name)
            self.assertIn('gh release download MatriX.145.Flow-v1.1.7 ', text)
            self.assertIn('6317ea5556bb6e70b7f31a024c6bcd9da2cf592789b67ab825fe0f3dc4c44bcf', text)
            self.assertIn('--source-digest 87d1632e47d64adcfd98d7d916d80aebedbc4e8c', text)
            self.assertIn('--source-ref refs/tags/MatriX.145.Flow-v1.1.7', text)
            self.assertIn('--deny-self-hosted-runners', text)
            self.assertNotIn('gh release download MatriX.145.Flow-v1.0.1 ', text)
            self.assertNotIn('gh release download MatriX.145.Flow-v1.1.3 ', text)

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

    def test_release_builds_share_one_validated_web_artifact_in_same_run(self):
        web = job('ts_release.yml', 'web')
        self.assertIn('uses: ./.github/workflows/build-web.yml', web)
        self.assertIn('artifact-name: flow-web-cross', web)
        for name in ('cross', 'macos'):
            caller = job('ts_release.yml', name)
            self.assertIn('needs: [classify, web]', caller)
            self.assertIn('web_artifact: flow-web-cross', caller)
        for workflow, name in [('build.yml', 'cross'), ('build-macos.yml', 'macos')]:
            consumer = job(workflow, name)
            # A failed local web build cannot become an accepted shared build.
            self.assertIn("needs.web.result == 'success'", consumer)
            self.assertNotIn("needs.web.result == 'skipped'", consumer)
            self.assertIn("name: ${{ inputs.web_artifact || 'flow-web-", consumer)
            self.assertNotIn('run-id:', consumer)

    def test_shared_web_keeps_successful_jobs_and_skips_only_validation_steps(self):
        for workflow in ('build.yml', 'build-macos.yml'):
            web = job(workflow, 'web')
            self.assertIn('reuse-artifact: ${{ inputs.web_artifact }}', web)
            self.assertNotIn('!inputs.web_artifact', web)
        web = job('build-web.yml', 'web')
        self.assertIn('Download validated same-run web UI', web)
        self.assertIn("if: inputs.reuse-artifact != ''", web)
        self.assertIn('if: ${{ !inputs.reuse-artifact }}', web)
        release = job('ts_release.yml', 'release')
        self.assertIn('python3 build/release_gates.py verified-jobs.json', release)

    def test_independent_checks_do_not_serialize_platform_builds(self):
        cross = job('build.yml', 'cross')
        self.assertIn('needs: [sources, web]', cross)
        self.assertNotIn('native-streaming-tests', cross)
        self.assertNotIn('go test ', cross)
        self.assertNotIn('playback_harness.py', cross)
        self.assertIn('name: flow-linux-native-deps', cross)
        native = job('build.yml', 'linux-validation')
        for gate in ('Native upload cache-miss regression', 'Unit tests', 'Go vet',
                     'Verify API contracts', 'Race tests and reachable Go vulnerability scan',
                     'Bounded fuzz campaigns', 'Library persistence measurement'):
            self.assertIn(gate, native)
        self.assertIn('name: flow-linux-native-deps', native)
        self.assertIn('tar -xzf _out/linux-native-deps.tar.gz', native)

    def test_every_integration_case_is_retained_and_dispatched(self):
        script = (ROOT / 'build/ci-integration.sh').read_text()
        expected = {
            'playback_harness.py': 1, 'startup_harness.py': 2,
            'startup_piece_check.py': 3, 'discovery_harness.py': 1,
            'preparation_harness.py': 1, 'peerhint_harness.py': 1,
            'webseed_harness.py': 1, 'adaptive_harness.py': 1,
            'request_capacity_check.py': 1, 'request_cancel_check.py': 1,
            'integrity_harness.py': 1, 'sparse_harness.py': 2,
            'maintenance_harness.py': 1, 'upstream_tls_check.py': 1,
            'console_harness.py': 1,
        }
        commands = re.findall(r'^\s*python3 build/([\w.]+)', script, re.M)
        for name, count in expected.items():
            self.assertEqual(commands.count(name), count, name)
        declared = set(re.findall(r'^    (\w+)\)', script, re.M))
        scheduled = set(re.search(r'group: \[([^]]+)\]',
                                 job('build.yml', 'controlled-integration'))[1].split(', '))
        self.assertEqual(declared, scheduled)
        for flag in ('--duration 20 --debug', '--profile custom --assert-fast',
                     '--case ordering --assert-fixed', '--case progress --assert-fixed',
                     '--case missing-index --assert-fixed',
                     '--rate-aware --cases below complementary late-have'):
            self.assertIn(flag, script)
        self.assertIn('set -euo pipefail', script)

    def test_publication_remains_gated_and_merged_receipts_remain_compatible(self):
        release = job('ts_release.yml', 'release')
        self.assertIn('needs: [classify, cross, macos]', release)
        self.assertNotIn('if: always()', release)
        self.assertIn('Attest release artifacts', release)
        self.assertIn('Verify reproducible packaging', release)
        for gate in ('native-streaming-tests', 'linux-validation', 'controlled-evidence',
                     'high-bitrate-comparison', 'windows-service', 'native-profiling'):
            self.assertIn(gate, job('build.yml', 'docker').split('if:')[0])
        evidence = job('build.yml', 'controlled-evidence')
        self.assertIn('needs: [linux-validation, controlled-integration]', evidence)
        self.assertIn('merge-multiple: true', evidence)
        self.assertIn('name: flow-controlled-playback', evidence)
        matrix = job('build.yml', 'high-bitrate-comparison')
        self.assertEqual(matrix.count('- {mbps:'), 10)
        self.assertIn("'5' || '3'", matrix)
        self.assertIn("'30' || '120'", matrix)

    def test_essential_scope_requires_unchanged_runtime_and_successful_full_run(self):
        classify = job('ts_release.yml', 'classify')
        self.assertIn('python3 build/essential_release.py d482e8c06d0e7cce7575f5300f30cfc70290153b', classify)
        self.assertIn('actions/runs/39761266311', classify)
        self.assertIn('d482e8c06d0e7cce7575f5300f30cfc70290153b\\tsuccess', classify)
        self.assertIn('scope=full', classify)
        self.assertIn('web_run_id=39761266311', classify)
        self.assertIn('run-id: ${{ steps.scope.outputs.web_run_id }}', classify)
        self.assertIn('--release-tag "$FLOW_TAG" --repository "$GITHUB_REPOSITORY" --github-output "$GITHUB_OUTPUT"', classify)
        self.assertIn('Make validated UI available in this run', classify)
        self.assertIn("needs.classify.outputs.validation_scope == 'essential'", job('ts_release.yml', 'web'))
        self.assertIn('if [ "$FLOW_TAG" = MatriX.145.Flow-v1.1.4 ]', classify)
        for name in ('native-profiling', 'high-bitrate-comparison'):
            self.assertIn("inputs.validation_scope != 'essential'", job('build.yml', name))
        for name in ('native-streaming-tests', 'linux-validation', 'controlled-integration',
                     'windows-service', 'release-inventory', 'windows-network-notifications'):
            self.assertNotIn('validation_scope', job('build.yml', name))

    def test_apt_keeps_signed_repositories_and_browser_cache_tracks_lockfile(self):
        script = (ROOT / 'build/ci-apt.sh').read_text()
        self.assertIn('https://archive.ubuntu.com/ubuntu', script)
        self.assertIn('Acquire::http::Timeout "20"', script)
        self.assertIn('Acquire::Retries "2"', script)
        for unsafe in ('--allow-unauthenticated', 'trusted=yes', 'Verify-Peer "false"'):
            self.assertNotIn(unsafe, script)
        web = job('build-web.yml', 'web')
        self.assertIn("hashFiles('web/yarn.lock')", web)
        self.assertIn('playwright install --with-deps chromium', web)
