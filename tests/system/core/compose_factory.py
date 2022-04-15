import os

from core.compose import DockerCompose


def create_docker_compose(env_vars=None) -> DockerCompose:
    script_dir = os.path.dirname(__file__)
    docker_compose_dir = os.path.dirname(script_dir)
    return DockerCompose(
        docker_compose_dir,
        compose_file_name=os.path.join(
            docker_compose_dir, "docker-compose.yaml"),
        project_name="stork_tests",
        env_vars=env_vars
    )
