import datetime
import glob
import os
from pathlib import Path
import re
import shutil
import sys
import time
import traceback
from typing import Any, List, Literal, Sequence, Tuple

import pytest

# In case of xdist the output is hidden by default.
# The redirection below forces output to screen.
if os.environ.get('PYTEST_XDIST_WORKER', False):
    sys.stdout = sys.stderr


def pytest_addoption(parser):
    parser.addoption("--stork-rpm-ver", action="store", help="Stork RPM packages version")
    parser.addoption("--stork-deb-ver", action="store", help="Stork deb packages version")
    group = parser.getgroup('selenium', 'selenium')
    group._addoption('--headless', action='store_true', help='Headless mode')


@pytest.fixture
def chrome_options(chrome_options, pytestconfig):
    if pytestconfig.getoption('headless'):
        chrome_options.add_argument('headless')
    return chrome_options


@pytest.fixture
def firefox_options(firefox_options, pytestconfig):
    if pytestconfig.getoption('headless'):
        firefox_options.set_headless(True)
    return firefox_options


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


def _get_pkg_version(pkg_pattern):
    """Get package version from package filename using provided pattern."""
    cwd = os.getcwd()
    if 'tests/system' in cwd:
        cwd = os.path.abspath(os.path.join(cwd, '../..'))
    paths = glob.glob(os.path.join(cwd, pkg_pattern))
    if len(paths) > 0 and len(paths) != 2:
        raise Exception('there are %d stork debs: %s' % (len(paths), str(paths)))
    elif len(paths) == 2:
        version_pattern = r'\d+\.\d+\.\d+\.\d+'
        vers = []
        for p in paths:
            m = re.search(version_pattern, p)
            if not m:
                raise Exception('cannot find version in %s' % paths[0])
            vers.append(m.group())
        if vers[0] != vers[1]:
            raise Exception('versions do not match %s' % str(vers))
        return vers[0]
    print('\n\nCannot find deb or rpm Stork packages.\nTo prepare them run `rake build_pkgs_in_docker`.\n')
    os._exit(1)

def pytest_configure(config):
    import containers

    # prepare packages versions: take them from option and if missing then detect them from package files
    containers.DEFAULT_STORK_RPM_VERSION = config.option.stork_rpm_ver
    if containers.DEFAULT_STORK_RPM_VERSION is None:
        ver = _get_pkg_version('isc-stork*rpm')
        containers.DEFAULT_STORK_RPM_VERSION = ver

    containers.DEFAULT_STORK_DEB_VERSION = config.option.stork_deb_ver
    if containers.DEFAULT_STORK_DEB_VERSION is None:
        ver = _get_pkg_version('isc-stork*deb')
        containers.DEFAULT_STORK_DEB_VERSION = ver

    # at the beginning clean up tests directory
    tests_dir = Path('test-results')
    if tests_dir.exists():
        shutil.rmtree(tests_dir)


def _prepare_containers(containers_to_create: Sequence[Tuple[str, str]]) \
        -> Sequence[Tuple[Literal['agent', 'server'], Any]]:
    """Build and run all necessary containers.
       Accepts sequence of tuples with container type (agent or server) and
       container operating system name.
       Return created, started containers."""
    import containers

    # change test case arguments from container system names
    # to actual started container instances
    server_containers: List[containers.StorkServerContainer] = []
    agent_containers: List[containers.StorkAgentContainer] = []
    # Must have a specific order
    # Items are tuples with container type and container object
    all_containers: List[Tuple[Literal['agent', 'server'], containers.Container]] = []  
    for name, val in containers_to_create:
        if name.startswith('agent'):
            a = containers.StorkAgentContainer(alias=val)
            agent_containers.append(a)
            all_containers.append(('agent', a))
        elif name.startswith('server'):
            s = containers.StorkServerContainer(alias=val)
            server_containers.append(s)
            all_containers.append(('server', s))
    assert len(server_containers) <= 1

    # start all agent containers in background so they can run in parallel and be ready quickly
    if server_containers:
        server_containers[0].setup_bg()
        while server_containers[0].mgmt_ip is None:
            time.sleep(0.1)

    for c in agent_containers:
        c.setup_bg(None, server_containers[0].mgmt_ip)

    # wait for all containers
    for _, c in all_containers:
        c.setup_wait()
        print('CONTAINER %s READY @ %s' % (c.name, c.mgmt_ip))

    time.sleep(3)

    return all_containers


@pytest.hookimpl(hookwrapper=True)
def pytest_pyfunc_call(pyfuncitem):
    """The purpose of this hook is:
    1. replacing test case system name arguments with container instances
    2. run the test case
    3. after testing collect logs from these containers
       and store them in `test-results` folder
    """
    # Prepare containers - build and run
    container_arguments = [arg for arg in pyfuncitem.funcargs.items()
                               if arg[0].startswith('server') or arg[0].startswith('agent')]

    try:
        containers = _prepare_containers(container_arguments)
    except Exception:
        print("ERROR: CANNOT PREPARE CONTAINERS")
        print(traceback.format_exc())
        pytest.skip("CANNOT PREPARE CONTAINERS")
        return

    # Assign containers to test arguments
    for (name, _), (_, container) in zip(container_arguments, containers):
        pyfuncitem.funcargs[name] = container

    try:
        # DO RUN TEST CASE
        outcome = yield
    finally:
        try:
            result = outcome.get_result()
        except Exception as ex:
            result = False

        print('TEST %s FINISHED: %s, COLLECTING LOGS' % (pyfuncitem.name, result))

        # prepare test directory for logs, etc
        tests_dir = Path('test-results')
        tests_dir.mkdir(exist_ok=True)
        test_name = pyfuncitem.name
        test_name = test_name.replace('[', '__')
        test_name = test_name.replace('/', '_')
        test_name = test_name.replace(']', '')
        test_dir = tests_dir / test_name
        if test_dir.exists():
            shutil.rmtree(test_dir)
        test_dir.mkdir()

        # download stork server and agent logs to test dir
        for idx, s in enumerate(c for t, c in containers if t == 'server'):
            _, out, _ = s.run('journalctl -u isc-stork-server')
            fname = test_dir / ('stork-server-%d.log' % idx)
            with open(fname, 'w') as f:
                f.write(out)
            s.stop()
        for idx, a in enumerate(c for t, c in containers if t == 'agent'):
            _, out, _ = a.run('journalctl -u isc-stork-agent')
            fname = test_dir / ('stork-agent-%d.log' % idx)
            with open(fname, 'w') as f:
                f.write(out)
            a.stop()
