import importlib.util
import json
from pathlib import Path
import tempfile
import unittest

spec = importlib.util.spec_from_file_location("flow_history_analyzer", Path(__file__).with_name("analyze_flow_history.py"))
analyzer = importlib.util.module_from_spec(spec)
spec.loader.exec_module(analyzer)


class HistoryAnalysisTests(unittest.TestCase):
    def test_latest_run_and_private_fields_are_not_exported(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "flow-history.jsonl"
            rows = [{"run": "old", "type": "first_byte", "elapsed_ms": 99999, "operation_measured": True},
                    {"run": "secret-token", "type": "first_byte", "elapsed_ms": 7000, "operation_measured": True},
                    {"run": "secret-token", "type": "peer_summary", "count": 2000},
                    {"run": "secret-token", "torrent": 1, "file": 3, "type": "playback_sample", "playback": {"session": 2, "buffer_seconds": 8, "title": "private-title", "ip": "private-ip"}},
                    {"run": "secret-token", "torrent": 1, "file": 3, "type": "playback_sample", "playback": {"session": 2, "buffer_seconds": 12}},
                    {"run": "secret-token", "type": "shutdown_requested", "stage": "SIGNAL"}]
            path.write_text("\n".join(json.dumps(row) for row in rows) + '\n{"partial":', encoding="utf-8")
            report = analyzer.summarize(path)
            self.assertEqual(report["response_start"]["maximum_ms"], 7000)
            self.assertEqual(report["peer_disconnects"], 2000)
            self.assertEqual(report["invalid_lines"], 1)
            self.assertEqual(report["shutdown_reason"], "SIGNAL")
            self.assertEqual(report["latest_delivery_samples"], [{"torrent": 1, "file": 3, "session": 2, "buffer_seconds": 12}])
            self.assertNotIn("private", json.dumps(report))
            self.assertNotIn("secret-token", json.dumps(report))

    def test_clean_closure_and_unknown_nested_enums(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "flow-history.jsonl"
            rows = [{"run": "x", "type": "startup_sample", "startup": {"blocked_on": "passkey=secret"}},
                    {"run": "x", "type": "engine_stopped"}]
            path.write_text("\n".join(map(json.dumps, rows)), encoding="utf-8")
            report = analyzer.summarize(path)
            self.assertEqual(report["closure"], "ENGINE_STOP_RECORDED")
            self.assertEqual(report["startup_blocked_samples"], {})
            self.assertNotIn("secret", json.dumps(report))

    def test_archive_rotation_preserves_latest_run(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory) / "flow-history.jsonl"
            Path(str(path) + ".1").write_text(json.dumps({"run": "x", "type": "startup_sample", "startup": {"blocked_on": "INDEX"}}), encoding="utf-8")
            path.write_text(json.dumps({"run": "x", "type": "engine_stopped"}), encoding="utf-8")
            self.assertEqual(analyzer.summarize(path)["startup_blocked_samples"], {"INDEX": 1})


if __name__ == "__main__":
    unittest.main()
