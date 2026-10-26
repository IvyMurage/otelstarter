#!/usr/bin/env python3
"""Compatibility check: does an app behave the same with otelstarter as without it?

For each app in compat/apps it:

1. sets the app up the way a user would: installs its dependencies, runs
   `otelstarter init --yes`, and installs what init asks for (the SDKs from
   this checkout, not from the registries);
2. runs it three times: plain, instrumented with a Collector, and
   instrumented with the Collector down;
3. sends the same requests each time and compares the answers;
4. stops it with SIGTERM and times the exit;
5. checks what reached the Collector: traces, logs and metrics, the route
   name, and no email address or password.

Usage (needs Go, Node.js 22.14+, Python 3.10+ and a Postgres in DATABASE_URL):

    DATABASE_URL=postgres://postgres:pw@127.0.0.1:5432/postgres python3 compat/run.py [app ...]

Standard library only, so CI needs nothing extra.
"""

from __future__ import annotations

import argparse
import gzip
import json
import os
import re
import shutil
import signal
import statistics
import subprocess
import sys
import threading
import time
import urllib.error
import urllib.request
from dataclasses import dataclass, field
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
APPS_DIR = ROOT / "compat" / "apps"
WORK = Path(os.environ.get("COMPAT_WORK", ROOT / "compat" / ".work"))
APP_PORT = 18080
COLLECTOR_PORT = 14318
DEAD_ENDPOINT = "http://127.0.0.1:9"  # nothing listens there: the Collector is down

# Leaks the Collector must never see: the SQL literal and the logged password.
SECRETS = [b"ann@shop.rw", b"hunter2"]
SHUTDOWN_LIMIT = 10.0  # seconds; Kubernetes' default grace period is 30


@dataclass
class App:
    name: str
    kind: str  # "node" or "python"
    start: list[str]  # {port} is replaced
    route: bytes  # the route template the framework reports, e.g. b"/orders/:id"
    src: str = ""  # folder in compat/apps, when shared with another entry
    build: list[str] = field(default_factory=list)

    @property
    def folder(self) -> str:
        return self.src or self.name


APPS = [
    App("express", "node", ["node", "server.js"], b"/orders/:id"),
    App("express-esm", "node", ["node", "server.js"], b"/orders/:id"),
    App(
        "nestjs11",
        "node",
        ["node", "dist/main.js"],
        b"/orders/:id",
        build=["npm", "run", "build"],
    ),
    App(
        "nestjs12",
        "node",
        ["node", "dist/main.js"],
        b"/orders/:id",
        build=["npm", "run", "build"],
    ),
    App("flask", "python", ["python", "app.py"], b"/orders/<int:order_id>"),
    App(
        "flask-gunicorn",
        "python",
        ["gunicorn", "-w", "2", "-b", "127.0.0.1:{port}", "app:app"],
        b"/orders/<int:order_id>",
        src="flask",
    ),
    App(
        "fastapi",
        "python",
        ["uvicorn", "main:app", "--port", "{port}"],
        b"/orders/{order_id}",
    ),
    App(
        "fastapi-workers",
        "python",
        ["uvicorn", "main:app", "--port", "{port}", "--workers", "2"],
        b"/orders/{order_id}",
        src="fastapi",
    ),
]

# (method, path, JSON body): the same in every run.
REQUESTS = [
    ("GET", "/health", None),
    ("GET", "/orders/42", None),
    ("GET", "/boom", None),
    ("GET", "/no-such-page", None),
    ("POST", "/echo", {"password": "hunter2", "n": 1}),
]


# --- a minimal OTLP/HTTP receiver ------------------------------------------


class Collector:
    """Accepts OTLP/HTTP exports and keeps the raw bodies. Strings are stored
    as-is in protobuf, so a byte search is enough to find a route or a leak."""

    def __init__(self, port: int) -> None:
        self.received: dict[str, list[bytes]] = {}
        self.lock = threading.Lock()
        collector = self

        class Handler(BaseHTTPRequestHandler):
            def do_POST(self) -> None:
                body = self.read_body()
                if self.headers.get("Content-Encoding") == "gzip":
                    body = gzip.decompress(body)
                signal_name = self.path.rsplit("/", 1)[-1]  # /v1/traces -> traces
                with collector.lock:
                    collector.received.setdefault(signal_name, []).append(body)
                self.send_response(200)
                self.send_header("Content-Type", "application/x-protobuf")
                self.send_header("Content-Length", "0")
                self.end_headers()

            def read_body(self) -> bytes:
                # Node's exporter sends the body in chunks, without Content-Length.
                if self.headers.get("Transfer-Encoding", "").lower() != "chunked":
                    return self.rfile.read(int(self.headers.get("Content-Length", 0)))
                body = b""
                while size := int(self.rfile.readline().split(b";")[0], 16):
                    body += self.rfile.read(size)
                    self.rfile.readline()  # the CRLF after each chunk
                self.rfile.readline()  # the CRLF after the last, empty chunk
                return body

            def log_message(self, *args: object) -> None:
                pass

        self.server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
        threading.Thread(target=self.server.serve_forever, daemon=True).start()

    def reset(self) -> None:
        with self.lock:
            self.received = {}

    def data(self, signal_name: str) -> bytes:
        with self.lock:
            return b"".join(self.received.get(signal_name, []))


# --- setting an app up -------------------------------------------------------


def sh(cmd: list[str], cwd: Path, env: dict[str, str] | None = None) -> str:
    result = subprocess.run(
        cmd, cwd=cwd, env=env, capture_output=True, text=True, check=False
    )
    if result.returncode != 0:
        raise SystemExit(
            f"$ {' '.join(cmd)}  (in {cwd})\n{result.stdout}\n{result.stderr}"
        )
    return result.stdout + result.stderr


def build_tools() -> tuple[Path, Path]:
    """Builds the CLI and packs the Node SDK from this checkout."""
    WORK.mkdir(parents=True, exist_ok=True)
    cli = WORK / "otelstarter"
    sh(["go", "build", "-o", str(cli), "./cmd/otelstarter"], ROOT)
    sdk = ROOT / "sdk" / "node"
    sh(["npm", "ci", "--no-audit", "--no-fund"], sdk)
    sh(["npm", "run", "build"], sdk)
    for old in WORK.glob("otelstarter-*.tgz"):
        old.unlink()
    sh(["npm", "pack", "--pack-destination", str(WORK)], sdk)
    tarball = next(WORK.glob("otelstarter-*.tgz"))
    return cli, tarball


def install_command(init_output: str) -> list[str]:
    """The install command init printed, e.g. "npm install --save-exact ..."."""
    for line in init_output.splitlines():
        line = line.strip()
        if line.startswith(("npm install", "pip install")):
            return line.split()
    return []


def prepare(app: App, cli: Path, tarball: Path, prepared: dict[str, Path]) -> Path:
    if app.folder in prepared:
        return prepared[app.folder]
    work = WORK / "apps" / app.folder
    shutil.rmtree(work, ignore_errors=True)
    shutil.copytree(APPS_DIR / app.folder, work)

    if app.kind == "node":
        sh(["npm", "install", "--no-audit", "--no-fund"], work)
        if app.build:
            sh(app.build, work)
        cmd = install_command(sh([str(cli), "init", "--yes", "."], work))
        # Use the SDK from this checkout instead of the published one.
        cmd = [
            str(tarball) if re.fullmatch(r"otelstarter@[\d.]+", c) else c for c in cmd
        ]
        if cmd:
            sh(cmd + ["--no-audit", "--no-fund"], work)
    else:
        sh([sys.executable, "-m", "venv", ".venv"], work)
        pip = [str(work / ".venv" / "bin" / "python"), "-m", "pip", "install", "-q"]
        sh(
            [str(cli), "init", "--yes", "."], work
        )  # adds the packages to requirements.txt
        # The SDK from this checkout first: it then satisfies init's
        # otelstarter==X pin, even before X is on PyPI.
        sh(pip + [str(ROOT / "sdk" / "python")], work)
        sh(pip + ["-r", "requirements.txt"], work)

    prepared[app.folder] = work
    return work


def read_env_file(path: Path) -> dict[str, str]:
    env = {}
    for line in path.read_text().splitlines():
        line = line.strip()
        if line and not line.startswith("#") and "=" in line:
            key, value = line.split("=", 1)
            env[key] = value
    return env


# --- running an app ----------------------------------------------------------


@dataclass
class Run:
    responses: list[tuple[int, object]] = field(default_factory=list)
    startup: float = 0.0
    rss_mb: float = 0.0
    latency_ms: float = 0.0
    shutdown: float = 0.0
    exit: str = ""
    output: str = ""
    error: str = ""


def request(method: str, path: str, body: object) -> tuple[int, object]:
    data = json.dumps(body).encode() if body is not None else None
    req = urllib.request.Request(
        f"http://127.0.0.1:{APP_PORT}{path}",
        data=data,
        method=method,
        headers={"Content-Type": "application/json"} if data else {},
    )
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            status, raw = resp.status, resp.read()
    except urllib.error.HTTPError as err:
        status, raw = err.code, err.read()
    try:
        return status, json.loads(raw)
    except ValueError:
        return status, raw.decode(errors="replace")


def group_rss_mb(pgid: int) -> float:
    total = 0
    for status in Path("/proc").glob("[0-9]*/status"):
        try:
            if os.getpgid(int(status.parent.name)) != pgid:
                continue
            match = re.search(r"VmRSS:\s+(\d+) kB", status.read_text())
            total += int(match.group(1)) if match else 0
        except (OSError, ValueError):
            continue
    return total / 1024


def run_app(app: App, work: Path, env: dict[str, str], command: list[str]) -> Run:
    run = Run()
    log = (work / f".run-{time.monotonic_ns()}.log").open("w+")
    started = time.monotonic()
    proc = subprocess.Popen(
        command,
        cwd=work,
        env=env,
        stdout=log,
        stderr=subprocess.STDOUT,
        start_new_session=True,
    )
    try:
        while time.monotonic() - started < 60:
            if proc.poll() is not None:
                run.error = f"exited during startup with {proc.returncode}"
                return run
            try:
                if request("GET", "/health", None)[0] == 200:
                    break
            except (OSError, urllib.error.URLError):
                time.sleep(0.1)
        else:
            run.error = "did not answer /health within 60 s"
            return run
        run.startup = time.monotonic() - started

        run.responses = [request(*r) for r in REQUESTS]
        timings = []
        for _ in range(100):
            t = time.monotonic()
            request("GET", "/orders/42", None)
            timings.append((time.monotonic() - t) * 1000)
        run.latency_ms = statistics.median(timings)
        run.rss_mb = group_rss_mb(proc.pid)

        t = time.monotonic()
        proc.send_signal(signal.SIGTERM)
        try:
            code = proc.wait(timeout=SHUTDOWN_LIMIT + 20)
            run.shutdown = time.monotonic() - t
            run.exit = (
                f"signal {signal.Signals(-code).name}" if code < 0 else f"code {code}"
            )
        except subprocess.TimeoutExpired:
            run.shutdown = time.monotonic() - t
            run.exit = "still running"
        return run
    finally:
        if proc.poll() is None:
            os.killpg(proc.pid, signal.SIGKILL)
            proc.wait()
        try:
            os.killpg(proc.pid, signal.SIGKILL)  # workers a master left behind
        except ProcessLookupError:
            pass
        log.seek(0)
        run.output = log.read()[-3000:]
        log.close()


# The real node binary. Version managers such as Volta put a launcher named
# node on PATH that runs node as a child process: SIGTERM would stop the
# launcher at once and leave the app running, which tests nothing.
NODE = (
    subprocess.run(
        ["node", "-p", "process.execPath"], capture_output=True, text=True, check=False
    ).stdout.strip()
    or "node"
)


def commands(
    app: App, work: Path, env_file: dict[str, str], endpoint: str | None
) -> tuple[list[str], dict[str, str]]:
    command = [c.replace("{port}", str(APP_PORT)) for c in app.start]
    if command[0] == "node":
        command[0] = NODE
    env = {**os.environ, "PORT": str(APP_PORT), "NODE_ENV": "production"}
    if app.kind == "python":
        bin_dir = work / ".venv" / "bin"
        env["PATH"] = f"{bin_dir}:{env['PATH']}"
        env["VIRTUAL_ENV"] = str(work / ".venv")
    if endpoint is None:
        return command, env

    env.update(env_file)
    env.update(
        {
            "OTEL_EXPORTER_OTLP_ENDPOINT": endpoint,
            # Export quickly, so a short run still sends everything.
            "OTEL_BSP_SCHEDULE_DELAY": "200",
            "OTEL_BLRP_SCHEDULE_DELAY": "200",
            "OTEL_METRIC_EXPORT_INTERVAL": "1000",
        }
    )
    if app.kind == "node":
        env["NODE_OPTIONS"] = (
            env.get("NODE_OPTIONS", "") + " --require otelstarter/register"
        ).strip()
    else:
        command = ["opentelemetry-instrument", *command]
    return command, env


# --- checks -------------------------------------------------------------------


def compare(plain: Run, other: Run) -> list[str]:
    problems = []
    if other.error:
        return [other.error]
    for (method, path, _), want, got in zip(REQUESTS, plain.responses, other.responses):
        if want != got:
            problems.append(f"{method} {path}: {got!r}, without otelstarter {want!r}")
    if other.exit == "still running" or other.shutdown > SHUTDOWN_LIMIT:
        problems.append(f"SIGTERM: took {other.shutdown:.1f} s ({other.exit})")
    return problems


def check_telemetry(app: App, collector: Collector) -> list[str]:
    problems = []
    if os.environ.get("COMPAT_DEBUG"):
        print(
            "    received:",
            {k: [len(b) for b in v] for k, v in collector.received.items()},
        )
    traces, logs, metrics = (
        collector.data("traces"),
        collector.data("logs"),
        collector.data("metrics"),
    )
    if not traces:
        problems.append("no traces received")
    elif app.route not in traces:
        problems.append(f"no span names the route {app.route.decode()}")
    if b"echo received" not in logs:
        problems.append("the app's log 'echo received' was not received")
    if not metrics:
        problems.append("no metrics received")
    for secret in SECRETS:
        for name, data in (("traces", traces), ("logs", logs), ("metrics", metrics)):
            if secret in data:
                problems.append(f"{secret.decode()} reached the Collector in {name}")
    return problems


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("apps", nargs="*", help="apps to check (default: all)")
    args = parser.parse_args()
    if "DATABASE_URL" not in os.environ:
        raise SystemExit("Set DATABASE_URL to a Postgres the apps can query.")
    unknown = set(args.apps) - {a.name for a in APPS}
    if unknown:
        raise SystemExit(f"Unknown app(s): {', '.join(sorted(unknown))}")
    selected = [a for a in APPS if not args.apps or a.name in args.apps]

    cli, tarball = build_tools()
    collector = Collector(COLLECTOR_PORT)
    prepared: dict[str, Path] = {}
    failures = 0
    rows = []

    for app in selected:
        print(f"\n== {app.name}", flush=True)
        work = prepare(app, cli, tarball, prepared)
        env_file = read_env_file(work / "otel.env")

        plain = run_app(app, work, *reversed(commands(app, work, env_file, None)))
        if plain.error:
            print(f"  ✗ the app itself does not run: {plain.error}\n{plain.output}")
            failures += 1
            continue

        collector.reset()
        instrumented = run_app(
            app,
            work,
            *reversed(
                commands(app, work, env_file, f"http://127.0.0.1:{COLLECTOR_PORT}")
            ),
        )
        if os.environ.get("COMPAT_DEBUG"):
            print(
                f"    instrumented: exit {instrumented.exit} after {instrumented.shutdown:.2f} s"
            )
        time.sleep(1)  # anything still in flight after exit
        down = run_app(
            app, work, *reversed(commands(app, work, env_file, DEAD_ENDPOINT))
        )

        results = {
            "same answers and clean exit, Collector up": compare(plain, instrumented),
            "telemetry received, nothing leaked": []
            if instrumented.error
            else check_telemetry(app, collector),
            "same answers and clean exit, Collector down": compare(plain, down),
        }
        for title, problems in results.items():
            print(f"  {'✗' if problems else '✓'} {title}")
            for problem in problems:
                print(f"      {problem}")
        if any(results.values()):
            failures += 1
            print(
                "    output with otelstarter:\n"
                + "\n".join(
                    "      " + l for l in instrumented.output.splitlines()[-25:]
                )
            )
        rows.append((app.name, plain, instrumented, down))

    print("\nCost (median of 100 requests; memory of all the app's processes):\n")
    print(
        f"  {'app':16} {'startup s':>15} {'memory MB':>15} {'request ms':>15} {'exit s (down)':>15}"
    )
    for name, plain, inst, down in rows:
        print(
            f"  {name:16} {plain.startup:6.2f} → {inst.startup:6.2f} {plain.rss_mb:6.0f} → {inst.rss_mb:6.0f}"
            f" {plain.latency_ms:6.2f} → {inst.latency_ms:6.2f} {plain.shutdown:6.2f} → {down.shutdown:6.2f}"
        )
    print(
        f"\n{len(selected) - failures} of {len(selected)} apps behave the same with otelstarter."
    )
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
