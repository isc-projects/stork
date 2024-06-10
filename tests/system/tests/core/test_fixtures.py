from typing import Tuple

from core.compose_factory import create_docker_compose
from core.fixtures import kea_parametrize
from core.wrappers import Kea, Server


@kea_parametrize(suppress_registration=True)
def test_kea_only_fixture(kea_service: Kea):
    assert kea_service.is_operational()
    assert kea_service.server is None

    compose = create_docker_compose()
    assert not compose.is_operational("server")
    # If the agent is fully operational, the metrics endpoint should be
    # available.
    assert kea_service.read_prometheus_metrics()


def test_kea_with_implicit_server_fixture(kea_service: Kea):
    assert kea_service.is_operational()
    assert kea_service.server.is_operational()


def test_kea_with_explicit_server_fixture(server_service: Server, kea_service: Kea):
    assert kea_service.is_operational()
    assert kea_service.server.is_operational()
    assert server_service.is_operational()


def test_kea_ha_pair_fixture(ha_pair_service: Tuple[Kea, Kea]):
    kea_first, kea_second = ha_pair_service
    assert kea_first.is_operational()
    assert kea_second.is_operational()
