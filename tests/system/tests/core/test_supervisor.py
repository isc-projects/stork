from unittest.mock import MagicMock

from core.supervisor import SupervisorService


def test_create_supervisor_service():
    # Arrange
    mock = MagicMock()

    # Act
    service = SupervisorService(mock, "foo")

    # Assert
    assert service is not None


def test_get_pid():
    # Arrange
    mock = MagicMock()
    mock.return_value = (0, "123\n", "")

    service = SupervisorService(mock, "foo")

    # Act
    pid = service.get_pid()

    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.args[0]
    assert pid == 123
    assert cmd == ["supervisorctl", "pid", "foo"]


def test_is_operational_ok():
    # Arrange
    mock = MagicMock()
    mock.return_value = (0, "OK\n", "")

    service = SupervisorService(mock, "foo")

    # Act
    operational = service.is_operational()

    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.args[0]
    assert operational
    assert cmd == ["supervisorctl", "status", "foo"]


def test_is_operational_fail():
    # Arrange
    mock = MagicMock()
    mock.return_value = (1, "STOPPED\n", "")

    service = SupervisorService(mock, "foo")

    # Act
    operational = service.is_operational()

    # Assert
    mock.assert_called_once()
    cmd = mock.call_args.args[0]
    assert not operational
    assert cmd == ["supervisorctl", "status", "foo"]


def test_restart():
    # Arrange
    mock = MagicMock()
    mock.side_effect = [(0, "", ""), (0, "OK\n", "")]

    service = SupervisorService(mock, "foo")

    # Act
    service.restart()

    # Assert
    assert mock.call_count == 2
    cmd = mock.call_args_list[0].args[0]
    assert cmd == ["supervisorctl", "restart", "foo"]
    cmd = mock.call_args_list[1].args[0]
    assert cmd == ["supervisorctl", "status", "foo"]


def test_reload():
    # Arrange
    mock = MagicMock()
    mock.side_effect = [(0, "", ""), (1, "STOPPED\n", ""), (0, "OK\n", "")]

    service = SupervisorService(mock, "foo")

    # Act
    service.reload()

    # Assert
    assert mock.call_count == 3
    cmd = mock.call_args_list[0].args[0]
    assert cmd == ["supervisorctl", "signal", "HUP", "foo"]
    cmd = mock.call_args_list[1].args[0]
    assert cmd == ["supervisorctl", "status", "foo"]


def test_interrupt():
    # Arrange
    mock = MagicMock()
    mock.side_effect = [(0, "", ""), (0, "OK\n", ""), (1, "STOPPED\n", "")]

    service = SupervisorService(mock, "foo")

    # Act
    service.interrupt()

    # Assert
    assert mock.call_count == 3
    cmd = mock.call_args_list[0].args[0]
    assert cmd == ["supervisorctl", "signal", "INT", "foo"]
    cmd = mock.call_args_list[1].args[0]
    assert cmd == ["supervisorctl", "status", "foo"]
