
import pytest


from core.compose_factory import create_docker_compose
from core.wrappers import Kea, Server
from core.utils import memoize


def test_create_compose():
    compose = create_docker_compose()
    assert compose is not None


def test_fetch_empty_logs():
    compose = create_docker_compose()
    stdout, stderr = compose.get_logs()
    assert stderr == ""
    assert stdout != ""


def test_server_instance():
    service_name = "server"
    compose = create_docker_compose()
    compose.start(service_name)
    compose.wait_for_operational(service_name)
    status, health = compose.get_service_status(service_name)
    assert status == "running"
    assert health == "healthy"
    compose.stop()


def test_server_fixture(server_service):
    assert server_service is not None


def test_kea_only_instance():
    service_name = "agent-kea"
    env_vars = {"STORK_SERVER_URL": ""}
    compose = create_docker_compose(env_vars=env_vars)
    compose.start(service_name)
    compose.wait_for_operational(service_name)
    status, health = compose.get_service_status(service_name)
    assert status == "running"
    assert health == "healthy"
    compose.stop()


@pytest.mark.parametrize("kea_service", [{"suppress_registration": True}], indirect=True)
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


def test_get_ip_address(server_service: Server):
    assert server_service.ip_address == "172.20.42.2"


def test_memoize():
    # Arrange
    class Foo:
        def __init__(self, bar):
            self._bar = bar
            self._call_count = 0

        @memoize
        def method(self, baz):
            self._call_count += 1
            return baz + self._bar

    bob = Foo(1)
    alice = Foo(2)

    # Act
    assert bob.method(3) == 4
    assert bob.method(3) == 4
    assert bob._call_count == 1

    assert alice.method(3) == 5
    assert alice.method(3) == 5
    assert alice._call_count == 1
