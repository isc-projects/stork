import os.path
from unittest.mock import MagicMock, patch

from core.compose_factory import create_docker_compose


def test_create_compose():
    compose = create_docker_compose()
    assert compose is not None


def test_create_compose_project_directory():
    compose = create_docker_compose()
    cmd = compose.docker_compose_command()
    idx = cmd.index("--project-directory")
    path = cmd[idx + 1]
    assert os.path.isabs(path)
    # Is parent of the current file?
    assert __file__.startswith(path)


def test_create_compose_fixed_project_name():
    compose = create_docker_compose()
    cmd = compose.docker_compose_command()
    idx = cmd.index("--project-name")
    name = cmd[idx + 1]
    assert name == "stork_tests"


def test_create_compose_single_compose_file():
    compose = create_docker_compose()
    cmd = compose.docker_compose_command()
    idx = cmd.index("-f")
    path = cmd[idx + 1]
    assert os.path.isabs(path)
    assert path.endswith("/docker-compose.yaml")


@patch("subprocess.run")
def test_create_compose_uses_environment_variables(patch):
    source_env_vars = dict(foo="1", bar="2")
    compose = create_docker_compose(env_vars=source_env_vars)
    compose.up()
    patch.assert_called_once()
    target_env_vars = patch.call_args.kwargs["env"]
    assert source_env_vars.items() <= target_env_vars.items()
