import unittest
from resource_endurance import convergence


class Resources(unittest.TestCase):
    def test_unknown_is_not_zero_or_acceptance(self):
        self.assertFalse(convergence([])['known'])
        samples = [{'memory':{'rss_available':False,'handles_available':False,'goroutines':10}} for _ in range(24)]
        self.assertFalse(convergence(samples)['known'])

    def test_sustained_growth_fails_but_bounded_allocator_plateau_passes(self):
        samples = [{'memory':{'rss_available':True,'handles_available':True,'rss_bytes':64*1024*1024,'handles':80,'goroutines':30}} for _ in range(40)]
        self.assertTrue(convergence(samples)['passed'])
        for row in samples[-10:]: row['memory']['handles'] = 500
        self.assertFalse(convergence(samples)['passed'])


if __name__ == '__main__': unittest.main()
