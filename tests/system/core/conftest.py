import pytest
from core.compose import DockerCompose
import core.wrappers as wrappers


def create_docker_compose(*service_names) -> DockerCompose:
    return DockerCompose(
        ".", build=True, project_directory="../../", service_names=service_names
    )


@pytest.fixture
def server():
    service_name = "server"
    with create_docker_compose(service_name) as compose:
        wrapper = wrappers.Server(compose, service_name)
        yield wrapper
