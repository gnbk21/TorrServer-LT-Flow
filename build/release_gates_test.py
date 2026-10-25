import unittest
from release_gates import GROUPS, REQUIRED, TARGETS, validate_jobs


class ReleaseGateTests(unittest.TestCase):
    def rows(self):
        names = [*REQUIRED, *('cross / '+t for t in TARGETS),
                 *('cross / controlled-integration ('+g+')' for g in GROUPS)]
        return [dict(id=i, name=name, conclusion='success') for i, name in enumerate(names)]

    def test_all_actual_essential_roles_are_required(self):
        rows = self.rows()
        self.assertEqual(validate_jobs(rows, 'essential'), len(rows))
        for index in range(len(rows)):
            with self.subTest(role=rows[index]['name']), self.assertRaises(ValueError):
                validate_jobs(rows[:index]+rows[index+1:], 'essential')

    def test_skipped_failed_cancelled_or_unfinished_gates_cannot_pass(self):
        for conclusion in ('skipped', 'failure', 'cancelled', None):
            rows = self.rows()
            rows[0]['conclusion'] = conclusion
            with self.subTest(conclusion=conclusion), self.assertRaises(ValueError):
                validate_jobs(rows, 'essential')

    def test_latest_attempt_wins_and_successful_previous_roles_can_be_reused(self):
        rows = self.rows()
        old = dict(rows[0], id=-1, conclusion='failure')
        validate_jobs([old, *rows], 'essential')
        bad = dict(rows[0], id=1000, conclusion='skipped')
        with self.assertRaises(ValueError): validate_jobs([*rows, bad], 'essential')

    def test_full_mode_requires_both_comparison_and_profiling_matrices(self):
        rows = self.rows()
        with self.assertRaises(ValueError): validate_jobs(rows, 'full')
        rows += [dict(id=100+i, name=f'cross / high-bitrate-comparison ({i})', conclusion='success') for i in range(10)]
        rows += [dict(id=200+i, name=f'cross / native-profiling / capture ({i})', conclusion='success') for i in range(2)]
        with self.assertRaises(ValueError): validate_jobs(rows, 'full')
        rows += [dict(id=300+i, name=f'cross / Parallel streaming ({i})', conclusion='success') for i in range(6)]
        validate_jobs(rows, 'full')
        rows[-1]['conclusion'] = 'skipped'
        with self.assertRaises(ValueError): validate_jobs(rows, 'full')
