"""Exercise real console output against isolated native server instances."""
import argparse
import json
import re
import socket
import ssl
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path


def port():
    with socket.socket() as listener:
        listener.bind(("127.0.0.1", 0))
        return listener.getsockname()[1]


def request(url, tls=False):
    # The HTTPS case intentionally uses this disposable server's generated cert.
    context = ssl._create_unverified_context() if tls else None
    opener = urllib.request.build_opener(urllib.request.ProxyHandler({}), urllib.request.HTTPSHandler(context=context))
    with opener.open(url, timeout=2) as response:
        return response.read()


def run(executable, output):
    output.mkdir(parents=True, exist_ok=False)
    reports = []
    cases = [
        ("auto", [], True, False),
        ("plain-disabled", ["--console", "plain", "--console-interval", "0"], False, False),
        ("legacy", ["--console", "off"], False, False),
        ("file-log", ["--logpath", str(output / "file.log")], False, False),
        ("https", ["--ssl", "--force-https", "--console-interval", "0"], False, True),
    ]
    for name, extra, heartbeat, tls in cases:
        state = output / name
        state.mkdir()
        http_port, tls_port = port(), port()
        argv = [str(executable), "--path", str(state), "--ip", "127.0.0.1", "--port", str(http_port)]
        if tls:
            argv += ["--sslport", str(tls_port)]
        if heartbeat:
            argv += ["--console-interval", "5"]
        argv += extra
        base = f"{'https' if tls else 'http'}://127.0.0.1:{tls_port if tls else http_port}"
        capture = output / f"{name}.log"
        with capture.open("wb") as stream:
            process = subprocess.Popen(argv, stdout=stream, stderr=subprocess.STDOUT, cwd=state)
            try:
                deadline = time.monotonic() + 40
                while True:
                    if process.poll() is not None:
                        raise RuntimeError(f"{name}: startup exit {process.returncode}; see {capture}")
                    try:
                        if request(base + "/flow/network", tls):
                            text = capture.read_text(encoding="utf-8", errors="replace")
                            if name in ("legacy", "file-log") or "QUICK HELP" in text:
                                break
                    except (OSError, urllib.error.URLError):
                        pass
                    if time.monotonic() > deadline:
                        raise RuntimeError(f"{name}: startup timeout")
                    time.sleep(0.1)
                if heartbeat:
                    time.sleep(5.5)
                request(base + "/shutdown", tls)
                process.wait(timeout=20)
                if process.returncode != 0:
                    raise RuntimeError(f"{name}: shutdown exit {process.returncode}")
            finally:
                if process.poll() is None:
                    process.kill()
                    process.wait(timeout=10)
        text = capture.read_text(encoding="utf-8", errors="replace")
        if "\x1b" in text:
            raise AssertionError(f"{name}: ANSI escapes in redirected output")
        if name in ("auto", "plain-disabled", "https"):
            for expected in ("TORRSERVER FLOW", "CONNECTION", "CONFIGURATION", "RUNTIME", "QUICK HELP", base + "/", "[Server] Stopped."):
                if expected not in text:
                    raise AssertionError(f"{name}: missing {expected!r}")
            if "LAN candidate" in text:
                raise AssertionError(f"{name}: loopback-only listener advertised LAN")
            if not re.search(r"\d\d:\d\d:\d\d INFO", text):
                raise AssertionError(f"{name}: missing readable log prefix")
        else:
            if "TORRSERVER FLOW |" in text or "[Status]" in text:
                raise AssertionError(f"{name}: console enabled for legacy/file mode")
        count = text.count("[Status] Uptime")
        if heartbeat and count < 2:
            raise AssertionError(f"{name}: missing heartbeat: {count}")
        if not heartbeat and count:
            raise AssertionError(f"{name}: disabled status emitted")
        if heartbeat:
            for expected in ("Streams 0", "cache resident", "Process RSS", "Go heap", "goroutines"):
                if expected not in text:
                    raise AssertionError(f"{name}: missing status field {expected}")
            if text.rfind("[Status]") > text.rfind("[Server] Stopped."):
                raise AssertionError("reporter wrote after shutdown")
        if name == "file-log":
            file_text = (output / "file.log").read_text(encoding="utf-8", errors="replace")
            if "UTC0" not in file_text or "\x1b" in file_text or "TORRSERVER FLOW |" in file_text or "[Status]" in file_text:
                raise AssertionError("file log compatibility failed")
        reports.append({"case": name, "passed": True, "status_reports": count})

    version = subprocess.run([str(executable), "--version"], capture_output=True, text=True, timeout=10, check=True)
    if not version.stdout.startswith("TorrServer-LT ") or "TORRSERVER FLOW |" in version.stdout:
        raise AssertionError("--version contract changed")
    doctor_state = output / "doctor"
    doctor_state.mkdir()
    doctor = subprocess.run([str(executable), "--doctor", "--path", str(doctor_state), "--ip", "127.0.0.1", "--port", str(port())], capture_output=True, text=True, timeout=10, check=True)
    if not isinstance(json.loads(doctor.stdout), list):
        raise AssertionError("doctor is not machine-readable JSON")
    reports.append({"case": "machine-readable-commands", "passed": True})

    for extra in (["--console", "invalid"], ["--console-interval", "1"]):
        invalid = subprocess.run([str(executable), *extra], capture_output=True, text=True, timeout=10)
        if invalid.returncode != 1 or "Flow console:" not in invalid.stderr:
            raise AssertionError("invalid console option not rejected")
    with socket.socket() as occupied:
        occupied.bind(("127.0.0.1", 0))
        occupied.listen()
        failed_state = output / "occupied"
        failed_state.mkdir()
        failed = subprocess.run([str(executable), "--path", str(failed_state), "--ip", "127.0.0.1", "--port", str(occupied.getsockname()[1])], capture_output=True, text=True, timeout=20)
        if failed.returncode != 1 or "[ERROR]" in failed.stdout or "ERROR [HTTP]" not in failed.stdout or "--port" not in failed.stdout or "TORRSERVER FLOW |" in failed.stdout:
            raise AssertionError(f"bind failure presentation: {failed.stdout}")
        (output / "occupied.log").write_text(failed.stdout, encoding="utf-8")
    reports.append({"case": "invalid-options-and-bind-failure", "passed": True})
    report = {"executable": str(executable), "passed": True, "cases": reports}
    (output / "report.json").write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    print(json.dumps(report, indent=2))


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--executable", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    arguments = parser.parse_args()
    run(arguments.executable.resolve(), arguments.output.resolve())
