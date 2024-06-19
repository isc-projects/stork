import os.path
from pathlib import Path
import shutil

import pytest

from core.compose_factory import create_docker_compose
from core import wrappers
from core.utils import setup_logger
from core import lease_generators
from core import performance_chart

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
    return pytest.mark.parametrize(
        fixture_name,
        [
            {
                "service_name": service_name,
                "suppress_registration": suppress_registration,
            }
        ],
        indirect=True,
    )


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


def ha_pair_parametrize(
    first_service_name="agent-kea-ha1",
    second_service_name="agent-kea-ha2",
    suppress_registration=False,
):
    """
    Helper for parametrizing the Kea fixture.

    Parameters
    ----------
    first_service_name : str, optional
        Name of docker-compose service of the first Kea instance, by default "agent-kea-ha1"
    second_service_name : str, optional
        Name of docker-compose service of the second Kea instance, by default "agent-kea-ha2"
    suppress_registration : bool, optional
        Suppress the Stork Agent registration in a server, by default False

    Returns
    -------
    _ParametrizeMarkDecorator
        the Pytest decorator ready to use
    """
    return pytest.mark.parametrize(
        "ha_pair_service",
        [
            {
                "first_service_name": first_service_name,
                "second_service_name": second_service_name,
                "suppress_registration": suppress_registration,
            }
        ],
        indirect=True,
    )


def bind9_parametrize(service_name="agent-bind9", suppress_registration=False):
    """
    Helper for parametrize the Bind9 fixture.

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
    return _agent_parametrize(
        "bind9_service",
        service_name,
        suppress_registration,
    )


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
    return pytest.mark.parametrize(
        "server_service", [{"service_name": service_name}], indirect=True
    )


def package_parametrize(version):
    """
    Sets the version of packages to install from the external repository.
    Empty version means the latest available.
    """
    return pytest.mark.parametrize(
        "package_service", [{"version": version}], indirect=True
    )


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
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)

    # Yield is used because we need to close the API connection even if any
    # error occurs.
    with wrappers.Server(compose, service_name) as wrapper:
        yield wrapper


@pytest.fixture
def kea_service(request):
    """
    A fixture setting up the Kea service and guarantees that it is
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
        "suppress_registration": False,
    }

    if hasattr(request, "param"):
        param.update(request.param)

    return _prepare_kea_wrapper(
        request=request,
        service_name=param["service_name"],
        suppress_registration=param["suppress_registration"],
    )


@pytest.fixture
def ha_pair_service(request):
    """
    A fixture setting up the Kea High-Availability pair services and
    guarantees that they are operational.

    Parameters
    ----------
    request : unknown
        Pytest request object

    Returns
    -------
    Tuple[core.wrappers.Kea]
        Kea wrappers for the docker-compose services

    Notes
    -----
    You can use the ha_pair_parametrize helper for configure the service.
    """
    param = {
        "first_service_name": "agent-kea-ha1",
        "second_service_name": "agent-kea-ha2",
        "suppress_registration": False,
    }

    if hasattr(request, "param"):
        param.update(request.param)

    first_wrapper = _prepare_kea_wrapper(
        request, param["first_service_name"], param["suppress_registration"]
    )
    second_wrapper = _prepare_kea_wrapper(
        request, param["second_service_name"], param["suppress_registration"], "kea-ha2"
    )
    return first_wrapper, second_wrapper


def _prepare_kea_wrapper(
    request,
    service_name: str,
    suppress_registration: bool,
    config_dirname="kea",
):
    """
    The helper function setting up the Kea Server service and guarantees that
    it is operational.

    Parameters
    ----------
    request : unknown
        Pytest request object
    service_name : str
        The compose service name
    suppress_registration : bool
        Indicates the registration in the server should be suppressed.
    config_dirname : str, optional
        The target directory for auto-generated configurations, by default "kea"

    Returns
    -------
    core.wrappers.Kea
        Kea wrapper for the docker-compose service
    """
    # Starts server service or suppresses registration
    env_vars = {}
    server_service_instance = None
    if suppress_registration:
        env_vars["STORK_SERVER_URL"] = ""
        env_vars["STORK_AGENT_LISTEN_PROMETHEUS_ONLY"] = "true"
    else:
        # We need the Server to perform the registration
        server_service_instance = request.getfixturevalue("server_service")

    # Re-generate the lease files
    config_dir = os.path.join(os.path.dirname(__file__), "../config", config_dirname)
    with open(os.path.join(config_dir, "kea-leases4.csv"), "wt", encoding="utf-8") as f:
        lease_generators.gen_dhcp4_lease_file(f)

    with open(os.path.join(config_dir, "kea-leases6.csv"), "wt", encoding="utf-8") as f:
        lease_generators.gen_dhcp6_lease_file(f)

    # Setup wrapper
    compose = create_docker_compose(extra_env_vars=env_vars)
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)
    wrapper = wrappers.Kea(compose, service_name, server_service_instance)

    if not suppress_registration:
        wrapper.wait_for_registration()

    return wrapper


@pytest.fixture
def bind9_service(request):
    """
    A fixture that sets up the Bind9 Server service and guarantees that it is
    operational.

    Parameters
    ----------
    request : unknown
        Pytest request object

    Returns
    -------
    core.wrappers.Bind9
        Bind9 wrapper for the docker-compose service

    Notes
    -----
    You can use the bind9_parametrize helper for configuring the service.
    """
    param = {
        "service_name": "agent-bind9",
        "suppress_registration": False,
    }

    if hasattr(request, "param"):
        param.update(request.param)

    # Starts server service or suppresses registration
    env_vars = None
    server_service_instance = None
    if param["suppress_registration"]:
        env_vars = {
            "STORK_SERVER_URL": "",
            "STORK_AGENT_LISTEN_PROMETHEUS_ONLY": "true",
        }
    else:
        # We need the Server to perform the registration
        server_service_instance = request.getfixturevalue("server_service")

    # Setup wrapper
    service_name = param["service_name"]
    compose = create_docker_compose(extra_env_vars=env_vars)
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)
    wrapper = wrappers.Bind9(compose, service_name, server_service_instance)

    if not param["suppress_registration"]:
        wrapper.wait_for_registration()

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


@pytest.fixture
def package_service(request):
    """
    A fixture that setup the Stork Server and Stork Agent services installed
    from the packages fetched from the external repository and guarantees that
    they are operational.
    """
    param = {"version": ""}

    if hasattr(request, "param"):
        param.update(request.param)

    env_vars = {"STORK_CLOUDSMITH_VERSION": param["version"]}

    compose = create_docker_compose(env_vars)
    service_name = "packages"
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)
    wrapper = wrappers.ExternalPackages(compose, service_name)
    wrapper.wait_for_registration()
    return wrapper


@pytest.fixture
def postgres_service():
    """
    A fixture that sets up the PostgreSQL service and guarantees that it is
    operational.

    Parameters
    ----------
    request : unknown
        Pytest request object

    Returns
    -------
    core.wrappers.Postgres
        PostgreSQL wrapper for the docker-compose service
    """
    service_name = "postgres"
    compose = create_docker_compose()
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)
    wrapper = wrappers.Postgres(compose, service_name)
    return wrapper


@pytest.fixture(autouse=True)
def finish(request):
    """Save all logs to file and down all used containers."""
    function_name = request.function.__name__

    def collect_logs(test_dir: Path):
        # Collect logs only for failed cases
        # If the test fails due to non-assertion error then the call status is
        # unavailable.
        if hasattr(request.node, "rep_call") and not request.node.rep_call.failed:
            return

        compose = create_docker_compose()
        service_names = compose.get_created_services()

        # Collect logs only for docker-compose services
        if len(service_names) == 0:
            return

        # prepare test directory for logs, etc
        test_dir.mkdir(exist_ok=True)

        # Collect logs
        stdout, stderr = compose.logs()

        # Write logs
        with open(test_dir / "stdout.log", "wt", encoding="utf-8") as f:
            f.write(stdout)

        with open(test_dir / "stderr.log", "wt", encoding="utf-8") as f:
            f.write(stderr)

        # Collect inspect for non-operational services
        has_non_operational_service = False
        for service_name in service_names:
            if compose.is_operational(service_name):
                continue
            has_non_operational_service = True
            inspect_stdout = compose.inspect_raw(service_name)
            filename = f"inspect-{service_name}.json"
            with open(test_dir / filename, "wt", encoding="utf-8") as f:
                f.write(inspect_stdout)

        if has_non_operational_service:
            # Collect service statuses
            ps_stdout = compose.ps()
            with open(test_dir / "ps.out", "wt", encoding="utf-8") as f:
                f.write(ps_stdout)

    def collect_metrics(test_dir: Path):
        test_dir.mkdir(exist_ok=True)

        compose = create_docker_compose()
        service_names = compose.get_created_services()

        report_paths = []
        for service_name in service_names:
            try:
                report_path = test_dir.resolve() / f"performance-report-{service_name}"
                compose.copy_to_host(
                    service_name, "/var/log/supervisor/performance-report", report_path
                )
                report_paths.append(report_path)
            except FileNotFoundError:
                # The container doesn't generate the performance report.
                pass

        if len(report_paths) != 0:
            performance_chart.plot_reports(
                report_paths, test_dir / "performance-charts.html"
            )

    def collect_logs_and_down_all():
        tests_dir = Path("test-results")
        tests_dir.mkdir(exist_ok=True)
        test_name = function_name
        test_name = test_name.replace("[", "__")
        test_name = test_name.replace("/", "_")
        test_name = test_name.replace("]", "")
        test_dir = tests_dir / test_name
        if test_dir.exists():
            shutil.rmtree(test_dir)

        # The result directory is not created yet.

        collect_logs(test_dir)
        collect_metrics(test_dir)
        # Down all containers
        compose = create_docker_compose()
        compose.down()

    request.addfinalizer(collect_logs_and_down_all)
