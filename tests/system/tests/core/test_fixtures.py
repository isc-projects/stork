from core.compose_factory import create_docker_compose
from core.fixtures import kea_parametrize
from core.wrappers import Kea, Server


@kea_parametrize(suppress_registration=True)
def test_kea_only_fixture(kea_service: Kea):
    assert kea_service.is_operational()
    assert kea_service.server is None

    compose = create_docker_compose()
    assert not compose.is_operational("server")


def test_kea_with_implicit_server_fixture(kea_service: Kea):
    assert kea_service.is_operational()
    assert kea_service.server.is_operational()


def test_kea_with_explicit_server_fixture(server_service: Server, kea_service: Kea):
    assert kea_service.is_operational()
    assert kea_service.server.is_operational()
    assert server_service.is_operational()
