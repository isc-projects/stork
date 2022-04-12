
import pytest


from core.compose_factory import create_docker_compose


def test_create_compose():
    compose = create_docker_compose()
    assert compose is not None


def test_fetch_empty_logs():
    compose = create_docker_compose()
    _, stderr = compose.get_logs()
    assert stderr == ""


def test_server_instance():
    service_name = "server"
    with create_docker_compose(service_name) as compose:
        compose.wait_for_healthy(service_name)
        status, health = compose.get_service_status(service_name)
        assert status == "running"
        assert health == "healthy"


def test_server_fixture(server_service):
    assert server_service is not None


def test_kea_instance():
    service_name = "agent-kea"
    env_vars = { "STORK_SERVER_URL": "" }
    with create_docker_compose(service_name, env_vars=env_vars) as compose:
        compose.wait_for_healthy(service_name)
        status, health = compose.get_service_status(service_name)
        assert status == "running"
        assert health == "healthy"


@pytest.mark.parametrize("kea_service", [{"suppress_registration": True}], indirect=True)
def test_kea_fixture(kea_service):
    assert kea_service is not None
