import os
from unittest.mock import patch, MagicMock

from core.compose_factory import create_docker_compose
from tests.core.commons import subprocess_result_mock, fake_compose_binary_detector


def test_create_compose():
    compose = create_docker_compose(compose_detector=fake_compose_binary_detector)
    assert compose is not None


def test_create_compose_project_directory():
    compose = create_docker_compose(compose_detector=fake_compose_binary_detector)
    cmd = compose.docker_compose_command()
    idx = cmd.index("--project-directory")
    path = cmd[idx + 1]
    assert os.path.isabs(path)
    # Is parent of the current file?
    assert __file__.startswith(path)


def test_create_compose_fixed_project_name():
    compose = create_docker_compose(compose_detector=fake_compose_binary_detector)
    cmd = compose.docker_compose_command()
    idx = cmd.index("--project-name")
    name = cmd[idx + 1]
    assert name == "stork_tests"


def test_create_compose_single_compose_file():
    compose = create_docker_compose(compose_detector=fake_compose_binary_detector)
    cmd = compose.docker_compose_command()
    idx = cmd.index("-f")
    path = cmd[idx + 1]
    assert os.path.isabs(path)
    assert path.endswith("/docker-compose.yaml")


@patch("subprocess.run")
def test_create_compose_uses_environment_variables(subprocess_run_mock: MagicMock):
    compose = create_docker_compose(
        extra_env_vars={"foo": "1", "bar": "2"},
        compose_detector=fake_compose_binary_detector,
        base_env_vars={"boz": "3"},
    )
    compose.up()
    subprocess_run_mock.assert_called_once()
    target_env_vars = subprocess_run_mock.call_args.kwargs["env"]

    assert "PWD" in target_env_vars
    del target_env_vars["PWD"]
    assert target_env_vars == {"foo": "1", "bar": "2", "boz": "3"}


@patch("subprocess.run", return_value=subprocess_result_mock(0, b"0.0.0.0:42080", b""))
def test_port_uses_localhost_instead_of_zero_host(subprocess_run_mock: MagicMock):
    compose = create_docker_compose(
        compose_detector=fake_compose_binary_detector, base_env_vars={}
    )
    address, _ = compose.port("server", 8080)
    subprocess_run_mock.assert_called_once()
    assert address == "localhost"


@patch("subprocess.run", return_value=subprocess_result_mock(0, b"foobar:42080", b""))
def test_port_preserves_custom_address(subprocess_run_mock: MagicMock):
    compose = create_docker_compose(compose_detector=fake_compose_binary_detector)
    address, _ = compose.port("server", 8080)
    subprocess_run_mock.assert_called_once()
    assert address == "foobar"


@patch("subprocess.run", return_value=subprocess_result_mock(0, b"0.0.0.0:42080", b""))
def test_port_uses_default_address_from_environment_variable(
    subprocess_run_mock: MagicMock,
):
    compose = create_docker_compose(
        compose_detector=fake_compose_binary_detector,
        base_env_vars={"DEFAULT_MAPPED_ADDRESS": "foobar"},
    )
    address, _ = compose.port("server", 8080)
    subprocess_run_mock.assert_called_once()
    assert address == "foobar"
