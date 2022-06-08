import os.path
from pathlib import Path
import shutil

import pytest

from core.compose_factory import create_docker_compose
import core.wrappers as wrappers
from core.utils import setup_logger
import core.lease_generators as lease_generators


logger = setup_logger(__name__)


def _agent_parametrize(fixture_name, service_name, suppress_registration=False):
    """
    Helper for parametrizing the agent fixtures.

    Parameters
    ----------
    fixture_name : str  
        Name of the Pytest fixture
    service_name : str
        Name of docker-compose service
    suppress_registration : bool, optional
        Suppress the Stork Agent registration in a server, by default False

    Returns
    -------
    _ParametrizeMarkDecorator
        the Pytest decorator ready to use
    """
    return pytest.mark.parametrize(fixture_name, [{
        "service_name": service_name,
        "suppress_registration": suppress_registration
    }], indirect=True)


def kea_parametrize(service_name="agent-kea", suppress_registration=False):
    """
    Helper for parametrizing the Kea fixture.

    Parameters
    ----------
    service_name : str, optional
        Name of docker-compose service of the Kea, by default "agent-kea"
    suppress_registration : bool, optional
        Suppress the Stork Agent registration in a server, by default False

    Returns
    -------
    _ParametrizeMarkDecorator
        the Pytest decorator ready to use
    """
    return _agent_parametrize("kea_service", service_name, suppress_registration)


def bind_parametrize(service_name="agent-bind9", suppress_registration=False):
    """
    Helper for parametrize the Bind 9 fixture.

    Parameters
    ----------
    service_name : str, optional
        Name of docker-compose service of the Kea, by default "agent-bind9"
    suppress_registration : bool, optional
        Suppress the Stork Agent registration in a server, by default False

    Returns
    -------
    _ParametrizeMarkDecorator
        the Pytest decorator ready to use
    """
    return _agent_parametrize("bind_service", service_name, suppress_registration)


def server_parametrize(service_name="server"):
    """
    Helper for parametrize the Stork Server fixture.

    Parameters
    ----------
    service_name : str, optional
        Name of docker-compose service of the Stork Server, by default "server"

    Returns
    -------
    _ParametrizeMarkDecorator
        the Pytest decorator ready to use
    """
    return pytest.mark.parametrize("server_service", [{
        "service_name": service_name
    }], indirect=True)


@pytest.fixture
def server_service(request):
    """
    A fixture that sets up the Stork Server service and guarantees that it is
    operational.

    Parameters
    ----------
    request : unknown
        Pytest request object

    Yields
    ------
    core.wrappers.Server
        Server wrapper for the docker-compose service

    Notes
    -----
    You can use the server_parametrize helper for configuring the service.
    """
    param = {
        "service_name": "server",
    }

    if hasattr(request, "param"):
        param.update(request.param)

    service_name = param["service_name"]

    compose = create_docker_compose()
    compose.start(service_name)
    compose.wait_for_operational(service_name)

    # Yield is used because we need to close the API connection even if any
    # error occurs.
    with wrappers.Server(compose, service_name) as wrapper:
        yield wrapper


@pytest.fixture
def kea_service(request):
    """
    A fixture that setup the Kea Server service and guarantees that it is
    operational.

    Parameters
    ----------
    request : unknown
        Pytest request object

    Returns
    -------
    core.wrappers.Kea
        Kea wrapper for the docker-compose service

    Notes
    -----
    You can use the kea_parametrize helper for configure the service.
    """
    param = {
        "service_name": "agent-kea",
        "suppress_registration": False
    }

    if hasattr(request, "param"):
        param.update(request.param)

    # Starts server service or suppresses registration
    env_vars = None
    server_service = None
    if param['suppress_registration']:
        env_vars = {"STORK_SERVER_URL": ""}
    else:
        # We need the Server to perform the registration
        server_service = request.getfixturevalue("server_service")

    # Re-generate the lease files
    config_dir = os.path.join(os.path.dirname(__file__), "../config/kea")
    with open(os.path.join(config_dir, "kea-leases4.csv"), "wt") as f:
        lease_generators.gen_dhcp4_lease_file(f)

    with open(os.path.join(config_dir, "kea-leases6.csv"), "wt") as f:
        lease_generators.gen_dhcp6_lease_file(f)

    # Setup wrapper
    service_name = param['service_name']
    compose = create_docker_compose(env_vars=env_vars)
    compose.start(service_name)
    compose.wait_for_operational(service_name)
    wrapper = wrappers.Kea(compose, service_name, server_service)
    return wrapper


@pytest.fixture
def bind_service(request):
    """
    A fixture that sets up the Bind8 Server service and guarantees that it is
    operational.

    Parameters
    ----------
    request : unknown
        Pytest request object

    Returns
    -------
    core.wrappers.Bind
        Bind wrapper for the docker-compose service

    Notes
    -----
    You can use the bind_parametrize helper for configuring the service.
    """
    param = {
        "service_name": "agent-bind9",
        "suppress_registration": False
    }

    if hasattr(request, "param"):
        param.update(request.param)

    # Starts server service or suppresses registration
    env_vars = None
    server_service = None
    if param['suppress_registration']:
        env_vars = {"STORK_SERVER_URL": ""}
    else:
        # We need the Server to perform the registration
        server_service = request.getfixturevalue("server_service")

    # Setup wrapper
    service_name = param['service_name']
    compose = create_docker_compose(env_vars=env_vars)
    compose.start(service_name)
    compose.wait_for_operational(service_name)
    wrapper = wrappers.Bind(compose, service_name, server_service)
    return wrapper


@pytest.fixture
def perfdhcp_service():
    """
    A fixture that allows controlling the perdhcp application.

    Returns
    -------
    core.wrappers.Perfdhcp
        Perfdhcp wrapper for the docker-compose service
    """
    service_name = "perfdhcp"
    compose = create_docker_compose()
    compose.build(service_name)
    wrapper = wrappers.Perfdhcp(compose, service_name)
    return wrapper


@pytest.fixture(autouse=True)
def finish(request):
    """Save all logs to file and down all used containers."""
    function_name = request.function.__name__

    def collect_logs_and_down_all():
        logger.info('COLLECTING LOGS')

        # Collect logs
        compose = create_docker_compose()
        stdout, stderr = compose.logs()

        # prepare test directory for logs, etc
        tests_dir = Path('test-results')
        tests_dir.mkdir(exist_ok=True)
        test_name = function_name
        test_name = test_name.replace('[', '__')
        test_name = test_name.replace('/', '_')
        test_name = test_name.replace(']', '')
        test_dir = tests_dir / test_name
        if test_dir.exists():
            shutil.rmtree(test_dir)
        test_dir.mkdir()

        # Write logs
        with open(test_dir / "stdout.log", 'wt') as f:
            f.write(stdout)

        with open(test_dir / "stderr.log", 'wt') as f:
            f.write(stderr)

        # Stop all containers
        compose.stop()
    request.addfinalizer(collect_logs_and_down_all)
