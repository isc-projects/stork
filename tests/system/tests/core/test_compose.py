import os
from unittest.mock import MagicMock

from core.compose_factory import create_docker_compose
from core.compose import DockerCompose


def test_command_contains_project_directory():
    compose = DockerCompose("project-dir")
    cmd = compose.docker_compose_command()
    assert "--project-directory project-dir" in " ".join(cmd)


def test_command_contains_project_name():
    compose = DockerCompose("project-dir", project_name="project-name")
    cmd = compose.docker_compose_command()
    assert "--project-name project-name" in " ".join(cmd)


def test_command_contains_default_project_name():
    compose = DockerCompose("parent/project-dir")
    cmd = compose.docker_compose_command()
    assert "--project-name project-dir" in " ".join(cmd)


def test_command_contains_environment_file():
    compose = DockerCompose("project-dir", env_file="env-file")
    cmd = compose.docker_compose_command()
    assert "--env-file env-file" in " ".join(cmd)


def test_command_contains_single_compose_file():
    compose = DockerCompose("project-dir", compose_file_name="compose.yaml")
    cmd = compose.docker_compose_command()
    assert "-f compose.yaml" in " ".join(cmd)


def test_command_contains_multiple_compose_file():
    compose = DockerCompose("project-dir",
                            compose_file_name=(
                                "compose1.yaml", "compose2.yaml"))
    cmd = compose.docker_compose_command()
    assert "-f compose1.yaml -f compose2.yaml" in " ".join(cmd)


def test_build_uses_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    compose.build()
    # Assert
    mock.assert_called_once()
    build_cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(build_cmd[:-1]) == " ".join(base_cmd)
    assert build_cmd[-1] == "build"


def test_build_uses_build_kit_by_default():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.build()
    # Assert
    env_vars = mock.call_args.kwargs["env_vars"]
    assert env_vars["DOCKER_BUILDKIT"] == "1"


def test_build_respects_build_kit_setting():
    # Arrange
    compose = DockerCompose("project-dir", use_build_kit=False)
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.build()
    # Assert
    env_vars = mock.call_args.kwargs["env_vars"]
    assert env_vars is None or "DOCKER_BUILDKIT" not in env_vars


def test_build_uses_service_names():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.build("foo", "bar")
    # Assert
    build_cmd = mock.call_args.kwargs["cmd"]
    assert build_cmd[-3] == "build"
    assert build_cmd[-2] == "foo"
    assert build_cmd[-1] == "bar"


def test_pull_uses_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    compose.pull()
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd[:-1]) == " ".join(base_cmd)
    assert cmd[-1] == "pull"


def test_pull_uses_service_names():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.pull("foo", "bar")
    # Assert
    build_cmd = mock.call_args.kwargs["cmd"]
    assert build_cmd[-3] == "pull"
    assert build_cmd[-2] == "foo"
    assert build_cmd[-1] == "bar"


def test_up_uses_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    compose.up()
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd[:-2]) == " ".join(base_cmd)
    assert cmd[-2] == "up"
    assert cmd[-1] == "-d"


def test_up_uses_service_names():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.up("foo", "bar")
    # Assert
    build_cmd = mock.call_args.kwargs["cmd"]
    assert build_cmd[-4] == "up"
    assert build_cmd[-2] == "foo"
    assert build_cmd[-1] == "bar"


def test_start_calls_only_up_by_default():
    # Arrange
    compose = DockerCompose("project-dir")
    pull_mock = MagicMock()
    build_mock = MagicMock()
    up_mock = MagicMock()
    compose.pull = pull_mock
    compose.build = build_mock
    compose.up = up_mock
    # Act
    compose.start()
    # Assert
    pull_mock.assert_not_called()
    build_mock.assert_not_called()
    up_mock.assert_called_once()


def test_start_can_call_build():
    # Arrange
    compose = DockerCompose("project-dir", build=True)
    pull_mock = MagicMock()
    build_mock = MagicMock()
    up_mock = MagicMock()
    compose.pull = pull_mock
    compose.build = build_mock
    compose.up = up_mock
    # Act
    compose.start()
    # Assert
    pull_mock.assert_not_called()
    build_mock.assert_called_once()
    up_mock.assert_called_once()


def test_start_can_call_pull():
    # Arrange
    compose = DockerCompose("project-dir", pull=True)
    pull_mock = MagicMock()
    build_mock = MagicMock()
    up_mock = MagicMock()
    compose.pull = pull_mock
    compose.build = build_mock
    compose.up = up_mock
    # Act
    compose.start()
    # Assert
    pull_mock.assert_called_once()
    build_mock.assert_not_called()
    up_mock.assert_called_once()


def test_start_uses_service_names():
    # Arrange
    compose = DockerCompose("project-dir", pull=True, build=True)
    pull_mock = MagicMock()
    build_mock = MagicMock()
    up_mock = MagicMock()
    compose.pull = pull_mock
    compose.build = build_mock
    compose.up = up_mock
    services = ("foo", "bar")
    # Act
    compose.start(*services)
    # Assert
    pull_mock.assert_called_once_with(*services)
    build_mock.assert_called_once_with(*services)
    up_mock.assert_called_once_with(*services)


def test_stop_calls_proper_command_and_removes_volumes():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    compose.stop()
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd[:-2]) == " ".join(base_cmd)
    assert cmd[-2] == "down"
    assert cmd[-1] == "-v"
