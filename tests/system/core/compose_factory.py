import os
from typing import Dict
import subprocess

from core.compose import DockerCompose
from core.constants import project_directory, docker_compose_file


def detect_compose_binary():
    """
    Detect a command to run the docker compose.
    The docker compose V1 is the standalone docker-compose executable.
    The docker compose V2 is a plugin to the docker core. It is available
    as subcommand: docker compose.
    The docker compose is end of life after June 2023 but it is still used
    in our CI systems.

    Returns
    -------
    list[str]
        The shell commands needed to run the docker compose.
    """
    commands = [["docker", "compose"], ["docker-compose"]]
    for command in commands:
        result = subprocess.run(command, check=False, capture_output=True)
        if result.returncode == 0:
            return command
    raise Exception("docker compose or docker-compose are not available")


def create_docker_compose(
    extra_env_vars: Dict[str, str] = None,
    compose_detector=detect_compose_binary,
    base_env_vars: Dict[str, str] = None,
) -> DockerCompose:
    """
    Creates the docker-compose controller that uses the system tests
    docker-compose file.

    The provided extra environment variables will be used in all system calls.
    The build arguments will be used in build calls.

    The docker-compose runs with a fixed project name to avoid duplicating the
    containers when the developer works with multiple project directories.

    If the docker-compose services aren't available on localhost, the valid
    hostname or IP address can be read from the DEFAULT_MAPPED_ADDRESS. It's
    helpful in Gitlab CI, where the Docker service is available under the
    "docker" hostname.

    The installed docker-compose version is detected using the provided detector.
    The default detector searches for executables in the system and prefers V2
    over V1.

    If the CS_REPO_ACCESS_TOKEN is set to non-empty value, the premium profile
    is enabled.

    The factory accepts the base environment dictionary to allow overriding the
    environment variables in the tests. If provided, it is used instead of the
    system environment variables.
    """
    profiles = []

    env_vars = base_env_vars if base_env_vars is not None else os.environ.copy()
    env_vars.update(extra_env_vars if extra_env_vars is not None else {})

    if env_vars.get("CS_REPO_ACCESS_TOKEN", "") != "":
        profiles.append("premium")

    return DockerCompose(
        project_directory,
        compose_file_name=docker_compose_file,
        project_name="stork_tests",
        env_vars=env_vars,
        build=True,
        default_mapped_hostname=env_vars.get("DEFAULT_MAPPED_ADDRESS", "localhost"),
        compose_base=compose_detector(),
        profiles=profiles,
    )
