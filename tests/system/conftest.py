import datetime
import os
import sys
from pathlib import Path
import shutil

import pytest as _

from core.fixtures import *
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
    # Stop all the running containers. The containers are stopped by default
    # on the testing end if no interruption happened
    compose = create_docker_compose()
    compose.stop()
    # Remove old test results
    tests_dir = Path('test-results')
    if tests_dir.exists():
        shutil.rmtree(tests_dir)
