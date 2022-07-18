import os
from typing import Dict

from core.compose import DockerCompose
from core.constants import project_directory, docker_compose_file


def create_docker_compose(env_vars: Dict[str, str] = None,
                          build_args: Dict[str, str] = None) -> DockerCompose:
    """
    Creates the docker-compose controller that uses the system tests
    docker-compose file.

    The provided environment variables will be used in all system calls. The
    build arguments will be used in build calls.

    The docker-compose runs with a fixed project name to avoid duplicating the
    containers when the developer works with multiple project directories.

    If the docker-compose services aren't available on localhost, the valid
    hostname or IP address can be read from the DEFAULT_MAPPED_ADDRESS. It's
    helpful in Gitlab CI, where the Docker service is available under the
    "docker" hostname.
    """
    return DockerCompose(
        project_directory,
        compose_file_name=docker_compose_file,
        project_name="stork_tests",
        env_vars=env_vars,
        build_args=build_args,
        build=True,
        default_mapped_hostname=os.environ.get(
            "DEFAULT_MAPPED_ADDRESS", "localhost"
        )
    )
