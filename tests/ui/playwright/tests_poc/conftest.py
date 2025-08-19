"""Test-level fixtures for Playwright PoC."""

import os
import subprocess
import pytest
from playwright.sync_api import Page
from tests.ui.playwright.pages.login_page import LoginPage


@pytest.fixture(scope="function")
def logged_in_page(page: Page):
    """Fixture to log in to Stork."""
    login = LoginPage(page)
    login.login("admin", "A123456a!")
    return page


def pytest_sessionfinish(session, exitstatus):
    """Cleanup Docker after all tests are done."""
    # pylint: disable=unused-argument
    env_path = os.path.abspath(".playwright_docker_env")
    if os.path.exists(env_path):
        with open(env_path, encoding="utf-8") as f:
            for line in f:
                if line.startswith("COMPOSE_PROJECT_NAME="):
                    project_name = line.strip().split("=")[1]
                    print(f"[PYTEST] Cleaning up Docker project: {project_name}")
                    subprocess.run(
                        [
                            "docker",
                            "compose",
                            "-p",
                            project_name,
                            "down",
                            "-v",
                            "--remove-orphans",
                        ],
                        check=True,
                    )
                    break
        os.remove(env_path)
