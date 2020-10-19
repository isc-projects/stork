import os
import re
import sys
import glob
import pytest

def pytest_addoption(parser):
    parser.addoption("--stork-rpm-ver", action="store", help="Stork RPM packages version")
    parser.addoption("--stork-deb-ver", action="store", help="Stork deb packages version")


def _get_pkg_version(pkg_pattern):
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

    containers.DEFAULT_STORK_RPM_VERSION = config.option.stork_rpm_ver
    if containers.DEFAULT_STORK_RPM_VERSION is None:
        ver = _get_pkg_version('isc-stork*rpm')
        containers.DEFAULT_STORK_RPM_VERSION = ver

    containers.DEFAULT_STORK_DEB_VERSION = config.option.stork_deb_ver
    if containers.DEFAULT_STORK_DEB_VERSION is None:
        ver = _get_pkg_version('isc-stork*deb')
        containers.DEFAULT_STORK_DEB_VERSION = ver
