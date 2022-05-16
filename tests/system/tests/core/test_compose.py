from unittest.mock import MagicMock, patch

from core.compose import ContainerExitedException, ContainerUnhealthyException, DockerCompose
from tests.core.commons import subprocess_result_mock


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
    assert " ".join(build_cmd).startswith(" ".join(base_cmd))
    assert "build" in build_cmd


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
    assert "build" in build_cmd
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


def test_start_calls_build():
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


def test_start_calls_pull():
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


def test_run_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    compose.run("service")
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    assert cmd[-1] == "service"
    assert "run" in cmd


def test_run_uses_arguments():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.run("service", "foo", "bar")
    # Assert
    cmd = mock.call_args.kwargs["cmd"]
    assert cmd[-3] == "service"
    assert cmd[-2] == "foo"
    assert cmd[-1] == "bar"


def test_run_checks_output_by_default():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.run("service")
    # Assert
    check = mock.call_args.kwargs["check"]
    assert check


def test_run_suppress_check_output():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.run("service", check=False)
    # Assert
    check = mock.call_args.kwargs["check"]
    assert not check


def test_logs_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "stdout", "stderr")
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    stdout, stderr = compose.logs()
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    # Has proper docker-compose general part?
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    # Calls the logs command?
    assert cmd[len(base_cmd)] == "logs"
    # Forces no colors?
    assert "--no-color" in cmd[len(base_cmd):]
    # Adds timestamps?
    assert "-t" in cmd[len(base_cmd):]
    # No more arguments
    assert len(cmd) == len(base_cmd) + 3
    # Checks output by default?
    assert "check" not in mock.call_args.kwargs
    # Has output?
    assert stdout == "stdout"
    assert stderr == "stderr"


def test_logs_uses_service_names():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "stdout", "stderr")
    compose._call_command = mock
    # Act
    compose.logs("service", "foo", "bar")
    # Assert
    cmd = mock.call_args.kwargs["cmd"]
    assert cmd[-2] == "foo"
    assert cmd[-1] == "bar"


def test_ps_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "stdout", "stderr")
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    stdout = compose.ps()
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    # Has proper docker-compose general part?
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    # Calls the ps command?
    assert cmd[len(base_cmd)] == "ps"
    # Includes all services?
    assert "--all" in cmd[len(base_cmd):]
    # No more arguments
    assert len(cmd) == len(base_cmd) + 2
    # Checks output by default?
    assert "check" not in mock.call_args.kwargs
    # Has output?
    assert stdout == "stdout"


def test_exec_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    compose.exec("service", ["command", "arg1", "arg2"])
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    assert "exec" in cmd[len(base_cmd):]
    assert "-T" in cmd[len(base_cmd):]
    assert cmd[-4] == "service"
    assert cmd[-3] == "command"
    assert cmd[-2] == "arg1"
    assert cmd[-1] == "arg2"
    assert mock.call_args.kwargs["check"]


def test_exec_suppress_check_output():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    compose._call_command = mock
    # Act
    compose.exec("service", ["command"], check=False)
    # Assert
    assert not mock.call_args.kwargs["check"]


def test_inspect_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.side_effect = [(0, "container-id", ""),
                        (0, "value-foo;value-bar", "")]
    compose._call_command = mock
    # Act
    result = compose.inspect("service", "foo", "bar")
    # Assert
    mock.assert_called()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith("docker inspect")
    assert cmd[-1] == "container-id"
    assert tuple(result) == ("value-foo", "value-bar")


def test_inspect_supports_none():
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.side_effect = [(0, "container-id", ""),
                        (0, "value-foo;<@NONE@>", "")]
    compose._call_command = mock
    # Act
    result = compose.inspect("service", "foo", "bar?")
    # Assert
    assert tuple(result) == ("value-foo", None)


def test_inspect_raw_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.side_effect = [(0, "container-id", ""),
                        (0, '{ "format": "json" }', "")]
    compose._call_command = mock
    # Act
    result = compose.inspect_raw("service")
    # Assert
    mock.assert_called()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith("docker inspect")
    assert cmd[-1] == "container-id"
    assert result == '{ "format": "json" }'


def test_port_calls_proper_command():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "0.0.0.0:1234", "")
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    result = compose.port("service", 80)
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    assert cmd[-2] == "service"
    assert cmd[-1] == "80"
    assert tuple(result) == ("0.0.0.0", 1234)


def test_port_replaces_default_address():
    # Arrange
    compose = DockerCompose("project-dir", default_mapped_hostname="foobar")
    mock = MagicMock()
    mock.return_value = (0, "0.0.0.0:1234", "")
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    result = compose.port("service", 80)
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    assert cmd[-2] == "service"
    assert cmd[-1] == "80"
    assert tuple(result) == ("foobar", 1234)


def test_get_service_ip_address_uses_proper_network_name():
    # Assert
    compose = DockerCompose("project-dir", project_name="prefix")
    mock = MagicMock()
    mock.return_value = ["123.45.67.89", ]
    compose.inspect = mock
    # Act
    ip = compose.get_service_ip_address("service", "network", family=4)
    # Assert
    service, fmt = mock.call_args.args
    assert ip == "123.45.67.89"
    assert service == "service"
    assert "prefix_network" in fmt


def test_get_container_id_for_existing_container():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "container-id", "")
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    container_id = compose.get_container_id("service")
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    assert tuple(cmd[-3:]) == ("ps", "-q", "service")
    assert container_id == "container-id"


def test_get_container_id_for_non_existing_container():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "", "")
    compose._call_command = mock
    base_cmd = compose.docker_compose_command()
    # Act
    container_id = compose.get_container_id("service")
    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.kwargs["cmd"]
    assert " ".join(cmd).startswith(" ".join(base_cmd))
    assert tuple(cmd[-3:]) == ("ps", "-q", "service")
    assert container_id is None


def test_get_service_status():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "running;healthy", "")
    compose._call_command = mock
    # Act
    status, health = compose.get_service_status("service")
    assert status == "running"
    assert health == "healthy"


def test_is_operational_for_running_healthy():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "running;healthy", "")
    compose._call_command = mock
    # Act & Assert
    assert compose.is_operational("service")


def test_is_not_operational_for_running_unhealthy():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "running;unhealthy", "")
    compose._call_command = mock
    # Act & Assert
    assert not compose.is_operational("service")


def test_is_not_operational_for_not_running_but_healthy():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "stopping;healthy", "")
    compose._call_command = mock
    # Act & Assert
    assert not compose.is_operational("service")


def test_is_operational_for_running_without_health():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "running;<@NONE@>", "")
    compose._call_command = mock
    # Act & Assert
    assert compose.is_operational("service")


def test_is_not_operational_for_not_running_without_health():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = (0, "stopping;<@NONE@>", "")
    compose._call_command = mock
    # Act & Assert
    assert not compose.is_operational("service")


def test_is_not_operational_for_unknown_container():
    # Arrange
    def side_effect(*args, **kwargs):
        raise LookupError()
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.side_effect = side_effect
    compose.get_container_id = mock
    # Act & Assert
    assert not compose.is_operational("service")


def test_get_created_services():
    # Arrange
    compose = DockerCompose("project-dir")
    call_command_mock = MagicMock()
    call_command_mock.return_value = (0, "foo\nbar\nbaz", "")
    compose._call_command = call_command_mock
    get_container_id_mock = MagicMock()
    get_container_id_mock.side_effect = ["1", None, "2"]
    compose.get_container_id = get_container_id_mock

    # Act
    services = compose.get_created_services()

    # Assert
    assert tuple(services) == ("foo", "baz")


def test_wait_for_operational_instant_success():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = ("running", "healthy")
    compose.get_service_status = mock
    # Act
    compose.wait_for_operational("service")
    # Assert
    assert mock.call_count == 1


def test_wait_for_operational_retry_to_success():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.side_effect = [("starting", "starting"),
                        ("running", "starting"), ("running", "healthy")]
    compose.get_service_status = mock
    # Act
    compose.wait_for_operational("service")
    # Assert
    assert mock.call_count == 3


def test_wait_for_operational_retry_to_success_without_health():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.side_effect = [("starting", None), ("running", None)]
    compose.get_service_status = mock
    # Act
    compose.wait_for_operational("service")
    # Assert
    assert mock.call_count == 2


def test_wait_for_operational_unhealthy():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = ("running", "unhealthy")
    compose.get_service_status = mock
    # Act
    try:
        compose.wait_for_operational("service")
    except ContainerUnhealthyException:
        # Assert
        pass


def test_wait_for_operational_exited():
    # Arrange
    compose = DockerCompose("project-dir")
    mock = MagicMock()
    mock.return_value = ("exited", None)
    compose.get_service_status = mock
    # Act
    try:
        compose.wait_for_operational("service")
    except ContainerExitedException:
        # Assert
        pass


@patch("subprocess.run")
def test_call_command_passes_command(patch: MagicMock):
    compose = DockerCompose("project-dir")
    compose._call_command(["foo", "bar"])
    patch.assert_called_once()
    cmd = patch.call_args.args[0]
    assert tuple(cmd) == ("foo", "bar")


@patch("subprocess.run")
def test_call_command_adds_env_vars(patch: MagicMock):
    compose = DockerCompose("project-dir", env_vars=dict(global_foo="1"))
    compose._call_command([], env_vars=dict(local_bar="2"))
    patch.assert_called_once()
    env = patch.call_args.kwargs["env"]
    assert env["global_foo"] == "1"
    assert env["local_bar"] == "2"
    assert env["PWD"] == "project-dir"
    # There is more variables from OS
    assert len(env) > 3


@patch("subprocess.run", return_value=subprocess_result_mock(0, b"foo\n", b"bar\n"))
def test_call_command_captures_output_by_default(patch: MagicMock):
    compose = DockerCompose("project-dir")
    status, stdout, stderr = compose._call_command([])
    patch.assert_called_once()
    item = patch.call_args.kwargs["capture_output"]
    assert item
    assert status == 0
    assert stdout == "foo"
    assert stderr == "bar"


@patch("subprocess.run", return_value=subprocess_result_mock(0, b"foo\n", b"bar\n"))
def test_call_command_suppreses_capturing_output(patch: MagicMock):
    compose = DockerCompose("project-dir")
    status, stdout, stderr = compose._call_command([], capture_output=False)
    patch.assert_called_once()
    item = patch.call_args.kwargs["capture_output"]
    assert not item
    assert status == 0
    assert stdout is None
    assert stderr is None


@patch("subprocess.run")
def test_call_command_checks_output_by_default(patch: MagicMock):
    compose = DockerCompose("project-dir")
    compose._call_command([])
    patch.assert_called_once()
    item = patch.call_args.kwargs["check"]
    assert item


@patch("subprocess.run")
def test_call_command_suppresses_checking_ouput(patch: MagicMock):
    compose = DockerCompose("project-dir")
    compose._call_command([], check=False)
    patch.assert_called_once()
    item = patch.call_args.kwargs["check"]
    assert not item


@patch("subprocess.run")
def test_call_sets_cwd_to_project_directory(patch: MagicMock):
    compose = DockerCompose("project-dir")
    compose._call_command([])
    patch.assert_called_once()
    cwd = patch.call_args.kwargs["cwd"]
    assert cwd == "project-dir"
