"""
Playwright UI tests against the SAME stack as system tests,
with a tiny Compose override that builds the full UI (server-ui target)
but keeps the service name 'server'.
"""

import os
import subprocess
import time
from pathlib import Path
from typing import List

import pytest
from playwright.sync_api import Page

# --- paths -------------------------------------------------------------------

CUR = Path(__file__).resolve()
TESTS_DIR = next(p for p in CUR.parents if p.name == "tests")
SYSTEM_DIR = TESTS_DIR / "system"
ROOT = TESTS_DIR.parent  # repo root

COMPOSE_BASE = str(SYSTEM_DIR / "docker-compose.yaml")
COMPOSE_UI = str(SYSTEM_DIR / "docker-compose.ui.yaml")

PROJECT_NAME = os.getenv("COMPOSE_PROJECT_NAME", "stork_tests")
BASE_URL = os.getenv("STORK_BASE_URL", "http://localhost:42080")


# --- helpers -----------------------------------------------------------------


def _dc_cmd(*args: str, capture: bool = False) -> subprocess.CompletedProcess:
    """Run docker compose with BOTH files loaded (base + UI override)."""
    cmd: List[str] = [
        "docker",
        "compose",
        "--ansi",
        "never",
        "--project-directory",
        str(ROOT),
        "-p",
        PROJECT_NAME,
        "-f",
        COMPOSE_BASE,
        "-f",
        COMPOSE_UI,
        *args,
    ]
    return subprocess.run(
        cmd,
        check=True,
        text=True,
        stdout=(subprocess.PIPE if capture else None),
        stderr=(subprocess.PIPE if capture else None),
        cwd=str(ROOT),
        env={**os.environ, "COMPOSE_DOCKER_CLI_BUILD": "1"},
    )


def _hard_cleanup() -> None:
    """Clean previous runs (containers, volumes, orphan networks)."""
    try:
        _dc_cmd("down", "--remove-orphans", "--volumes")
    except subprocess.CalledProcessError as err:
        print(f"[cleanup] Encountered error during docker cleanup: {err}")
    subprocess.run(["docker", "network", "prune", "-f"], check=False)


def _wait_http_ok(url: str, timeout: float = 90.0) -> None:
    """Wait until a URL returns HTTP 200."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            cp = subprocess.run(
                ["curl", "-sS", "-o", "/dev/null", "-w", "%{http_code}", url],
                check=True,
                text=True,
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
            )
            if cp.stdout.strip() == "200":
                return
        except subprocess.CalledProcessError as err:
            print(f"[wait_http_ok] curl error while probing {url}: {err}")
        time.sleep(1)
    raise RuntimeError(f"Timeout waiting for {url}")


def _reset_db_and_server(base_url: str) -> None:
    """Drop & recreate DB schema, restart server, wait until healthy."""
    pg = f"{PROJECT_NAME}-postgres-1"
    srv = f"{PROJECT_NAME}-server-1"

    sql = "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
    subprocess.run(
        [
            "docker",
            "exec",
            "-i",
            pg,
            "psql",
            "-U",
            "stork",
            "-d",
            "stork",
            "-v",
            "ON_ERROR_STOP=1",
            "-c",
            sql,
        ],
        check=True,
        text=True,
    )

    subprocess.run(["docker", "restart", srv], check=True, text=True)

    _wait_http_ok(f"{base_url}/api/version", timeout=120)

    try:
        _dc_cmd("run", "--no-deps", "register", "register", "--non-interactive")
    except subprocess.CalledProcessError as err:
        print(
            f"[reset] 'register' helper failed; continuing for UI tests. Error: {err}"
        )


# --- pytest fixtures ---------------------------------------------------------


@pytest.fixture(scope="session")
def setup() -> None:
    """
    SAME environment as system tests, with UI assets enabled via override file.

    Workflow:
      - If STORK_REUSE=1: just wait for health (reuse an already-running stack).
      - Else: build and start postgres, server, agent-kea; then try registering.

    Note: This fixture performs environment setup once per test session.
    Tests should use BASE_URL directly for the URL; this fixture returns nothing.
    """

    os.environ.setdefault("IPWD", str(ROOT))
    os.environ.setdefault("DOCKER_DEFAULT_PLATFORM", "linux/amd64")

    if os.getenv("STORK_REUSE") == "1":
        _wait_http_ok(f"{BASE_URL}/api/version", timeout=120)
        return

    _hard_cleanup()

    _dc_cmd("build", "--", "postgres", "server", "agent-kea")
    _dc_cmd("up", "-d", "--", "postgres")
    _dc_cmd("up", "-d", "--", "server")
    _dc_cmd("up", "-d", "--", "agent-kea")

    _wait_http_ok(f"{BASE_URL}/api/version", timeout=120)

    try:
        _dc_cmd("run", "--no-deps", "register", "register", "--non-interactive")
    except subprocess.CalledProcessError as e:
        print("WARN: 'register' helper failed; continuing for UI tests.\n", e)


@pytest.fixture(scope="function")
def logged_in_page(page: Page, setup):
    """Open login and authenticate with seeded admin credentials."""
    from tests.ui.playwright.pages.login_page import LoginPage

    lp = LoginPage(page)
    lp.open(BASE_URL)
    lp.login("admin", "admin")
    return page


@pytest.fixture()
def clean_env(setup):
    """Reset DB + restart server only when explicitly requested by a test."""
    _reset_db_and_server(BASE_URL)


@pytest.fixture(scope="session")
def base_url():
    return BASE_URL
