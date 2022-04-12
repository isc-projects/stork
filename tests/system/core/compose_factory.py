import os

from core.compose import DockerCompose


def create_docker_compose(*service_names, env_vars=None) -> DockerCompose:
    script_dir = os.path.dirname(__file__)
    docker_compose_dir = os.path.dirname(script_dir)
    project_directory = os.path.dirname(os.path.dirname(docker_compose_dir))
    return DockerCompose(
        project_directory,
        compose_file_name=os.path.join(docker_compose_dir, "docker-compose.yaml"),
        env_vars=env_vars,
        service_names=service_names
    )