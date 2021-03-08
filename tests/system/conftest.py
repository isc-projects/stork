import os
import re
import sys
import time
import glob
import shutil
from pathlib import Path

import pytest


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


@pytest.hookimpl(hookwrapper=True)
def pytest_pyfunc_call(pyfuncitem):
    """The purpose of this hook is:
    1. replacing test case system name arguments with container instances
    2. run the test case
    3. after testing collect logs from these containers
       and store them in `test-results` folder
    """
    import containers

    # change test case arguments from container system names
    # to actual started container instances
    srv_cntrs = []
    agn_cntrs = []
    for name, val in pyfuncitem.funcargs.items():
        if name.startswith('agent'):
            a = containers.StorkAgentContainer(alias=val)
            agn_cntrs.append(a)
            pyfuncitem.funcargs[name] = a
        elif name.startswith('server'):
            s = containers.StorkServerContainer(alias=val)
            srv_cntrs.append(s)
            pyfuncitem.funcargs[name] = s
    assert len(srv_cntrs) <= 1
    all_cntrs = srv_cntrs + agn_cntrs

    # start all agent containers in background so they can run in parallel and be ready quickly
    if srv_cntrs:
        srv_cntrs[0].setup_bg()
        while srv_cntrs[0].mgmt_ip is None:
            time.sleep(0.1)

    for c in agn_cntrs:
        c.setup_bg(None, srv_cntrs[0].mgmt_ip)

    # wait for all containers
    for c in all_cntrs:
        c.setup_wait()
        print('CONTAINER %s READY @ %s' % (c.name, c.mgmt_ip))
    time.sleep(3)

    # DO RUN TEST CASE
    outcome = yield

    print('TEST %s FINISHED: %s, COLLECTING LOGS' % (pyfuncitem.name, outcome.get_result()))

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
    for idx, s in enumerate(srv_cntrs):
        _, out, _ = s.run('journalctl -u isc-stork-server')
        fname = test_dir / ('stork-server-%d.log' % idx)
        with open(fname, 'w') as f:
            f.write(out)
        s.stop()
    for idx, a in enumerate(agn_cntrs):
        _, out, _ = a.run('journalctl -u isc-stork-agent')
        fname = test_dir / ('stork-agent-%d.log' % idx)
        with open(fname, 'w') as f:
            f.write(out)
        a.stop()
