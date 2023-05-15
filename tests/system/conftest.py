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
from core.fixtures import *  # noqa: F401, F403
from core.compose_factory import create_docker_compose

# In case of xdist the output is hidden by default.
# The redirection below forces output to screen.
if os.environ.get('PYTEST_XDIST_WORKER', False):
    sys.stdout = sys.stderr


def pytest_runtest_logstart(nodeid, location):
    banner = '\n\n************ START   %s ' % nodeid
    banner += '*' * (140 - len(banner))
    banner += '\n'
    banner = '\u001b[36m' + banner + '\u001b[0m'
    print(banner)


def pytest_runtest_logfinish(nodeid, location):
    banner = '\n************ END   %s ' % nodeid
    banner += '*' * (140 - len(banner))
    banner = '\u001b[36;1m' + banner + '\u001b[0m'
    print(banner)


def pytest_runtest_logreport(report):
    if report.when == 'call':
        dt = datetime.timedelta(seconds=int(report.duration))
        banner = '\n************ RESULT %s   %s  took %s  ' % (
            report.outcome.upper(), report.nodeid, dt)
        banner += '*' * (140 - len(banner))
        if report.outcome == 'passed':
            banner = '\u001b[32;1m' + banner + '\u001b[0m'
        else:
            banner = '\u001b[31;1m' + banner + '\u001b[0m'
        print(banner)


@pytest.hookimpl(tryfirst=True, hookwrapper=True)
def pytest_runtest_makereport(item, call):
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


def pytest_sessionstart(session):
    """
    Stop all the running containers. The containers are stopped by default
    on the testing end if no interruption happened
    """
    compose = create_docker_compose()
    compose.down()
    # Remove old test results
    tests_dir = Path('test-results')
    if tests_dir.exists():
        shutil.rmtree(tests_dir)


def pytest_collection_modifyitems(session, config, items):
    """
    The hook to add additional decorators/markers to the tests.
    """
    # Add per-test timeout.
    default_timeout = datetime.timedelta(minutes=20)

    for item in items:
        print(item)
        if item.get_closest_marker('timeout') is None:
            item.add_marker(pytest.mark.timeout(default_timeout.total_seconds()))

    # Skip all tests using the disabled services.
    compose = create_docker_compose()
    skip = pytest.mark.skip(reason="Skip due to not set the docker-compose profile.")

    for item in items:
        if not hasattr(item, "callspec"):
            continue
        callspec = item.callspec
        for _, fixture_args in callspec.params.items():
            service_name = fixture_args.get("service_name")
            if service_name is None:
                continue
            if not compose.is_enabled(service_name):
                item.add_marker(skip)
                break
