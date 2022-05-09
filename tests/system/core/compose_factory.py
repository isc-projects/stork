import os

from core.compose import DockerCompose
from core.constants import project_directory, docker_compose_dir


def create_docker_compose(env_vars=None) -> DockerCompose:
    return DockerCompose(
        project_directory,
        compose_file_name=os.path.join(
            docker_compose_dir, "docker-compose.yaml"),
        project_name="stork_tests",
        env_vars=env_vars,
        build=True,
        default_mapped_hostname=os.environ.get("DEFAULT_MAPPED_ADDRESS", "localhost")
    )
