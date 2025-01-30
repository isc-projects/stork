import datetime
import os
import sys
from pathlib import Path
import shutil

import pytest

# The F401 (module imported but unused) and F403 ('from module import *' used; unable to detect undefined names)
# Flake8 warnings are suppressed. We want to import all fixtures, so people implementing new fixtures according
# to python docs would have their lives easier and not figure out why it's not working. The F401 warning seems
# a bit bogus, anyway.
# pylint: disable=unused-import
from core.fixtures import (  # noqa: F401
    server_service,
    kea_service,
    ha_pair_service,
    bind9_service,
    perfdhcp_service,
    package_service,
    postgres_service,
    register_service,
    finish,
)
from core.compose_factory import create_docker_compose

# List of fixtures that run the Kea executables.
KEA_FIXTURE_NAMES = set(
    [
        kea_service.__name__,
        ha_pair_service.__name__,
        perfdhcp_service.__name__,
        register_service.__name__,
    ]
)

# In case of xdist the output is hidden by default.
# The redirection below forces output to screen.
if os.environ.get("PYTEST_XDIST_WORKER", False):
    sys.stdout = sys.stderr


def pytest_runtest_logstart(nodeid, location):  # pylint: disable=unused-argument
    """Called at the start of running the runtest protocol for a single item."""
    banner = f"\n\n************ START   {nodeid} "
    banner += "*" * (140 - len(banner))
    banner += "\n"
    banner = f"\u001b[36m{banner}\u001b[0m"
    print(banner)


def pytest_runtest_logfinish(nodeid, location):  # pylint: disable=unused-argument
    """Called at the end of running the runtest protocol for a single item."""
    banner = f"\n************ END   {nodeid} "
    banner += "*" * (140 - len(banner))
    banner = f"\u001b[36;1m{banner}\u001b[0m"
    print(banner)


def pytest_runtest_logreport(report):
    """Process the TestReport produced for each of the setup, call and teardown runtest phases of an item."""
    if report.when == "call":
        duration = datetime.timedelta(seconds=int(report.duration))
        banner = f"\n************ RESULT {report.outcome.upper()}   {report.nodeid}  took {duration}  "
        banner += "*" * (140 - len(banner))
        if report.outcome == "passed":
            banner = f"\u001b[32;1m{banner}\u001b[0m"
        else:
            banner = f"\u001b[31;1m{banner}\u001b[0m"
        print(banner)


@pytest.hookimpl(tryfirst=True, hookwrapper=True)
def pytest_runtest_makereport(item, call):  # pylint: disable=unused-argument
    """
    Making test result information available in fixtures
    Source: https://docs.pytest.org/en/latest/example/simple.html#making-test-result-information-available-in-fixtures
    """
    # execute all other hooks to obtain the report object
    outcome = yield
    rep = outcome.get_result()

    # set a report attribute for each phase of a call, which can
    # be "setup", "call", "teardown"

    setattr(item, "rep_" + rep.when, rep)


def pytest_sessionstart(session):  # pylint: disable=unused-argument
    """
    Stop all the running containers. The containers are stopped by default
    on the testing end if no interruption happened
    """
    compose = create_docker_compose()
    compose.down()
    # Remove old test results
    tests_dir = Path("test-results")
    if tests_dir.exists():
        shutil.rmtree(tests_dir)
    os.makedirs(tests_dir)

    perf_dir = Path("performance-results")
    if perf_dir.exists():
        shutil.rmtree(perf_dir)
    os.makedirs(perf_dir)


def is_debugger_active():
    """
    Return if the debugger is currently active.
    Source: https://stackoverflow.com/a/67065084
    """
    return hasattr(sys, "gettrace") and sys.gettrace() is not None


def pytest_collection_modifyitems(
    session, config, items
):  # pylint: disable=unused-argument
    """
    The hook to add additional decorators/markers to the tests.
    """
    # Add per-test timeout.
    # It is a watchdog to interrupt the test case if it is stuck (e.g. while
    # building the containers). Otherwise, it would wait until it is
    # interrupted by the global CI timeout, but in this case, the log and
    # diagnostic data wouldn't be collected.
    # We cannot set the timeout dynamically, depending if the test case needs
    # to build the Docker containers (longer period) or not (shorter period)
    # because there is no (built-in) way to detect if the build is necessary
    # before starting the test and adjusting timeout after it starts.
    #
    # The timeout is not set if the debugger is currently attached to the
    # system tests. It prevents to interrupt the debugging session.
    if not is_debugger_active():
        default_timeout = datetime.timedelta(minutes=20)
        for item in items:
            if item.get_closest_marker("timeout") is None:
                item.add_marker(pytest.mark.timeout(default_timeout.total_seconds()))

    compose = create_docker_compose()

    # Skip tests using the disabled services due to missing docker-compose
    # profile.
    skip_missing_tag = pytest.mark.skip(
        reason="Skip due to not set the docker-compose profile."
    )
    # Skip tests non-Kea related tests if the Kea-only environment variable is
    # set.
    skip_kea_only = pytest.mark.skip(reason="Skip because test is not related to Kea.")
    execute_only_kea_tests = os.environ.get("ONLY_KEA_TESTS", "false").lower() == "true"

    for item in items:
        if execute_only_kea_tests and not any(
            n in KEA_FIXTURE_NAMES for n in item.fixturenames
        ):
            item.add_marker(skip_kea_only)
            continue
        if not hasattr(item, "callspec"):
            # Only test cases with a parametrize decorator have callspec.
            continue
        callspec = item.callspec
        for _, fixture_args in callspec.params.items():
            service_name = fixture_args.get("service_name")
            if service_name is not None and not compose.is_enabled(service_name):
                item.add_marker(skip_missing_tag)
                break
