import datetime
import os
import sys

import pytest as _

from core.fixtures import *

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
        banner = '\n************ RESULT %s   %s  took %s  ' % (report.outcome.upper(), report.nodeid, dt)
        banner += '*' * (140 - len(banner))
        if report.outcome == 'passed':
            banner = '\u001b[32;1m' + banner + '\u001b[0m'
        else:
            banner = '\u001b[31;1m' + banner + '\u001b[0m'
        print(banner)
