from unittest.mock import MagicMock


def subprocess_result_mock(status, stdout, stderr):
    """Returns mock compatible with the subprocess run output."""
    mock = MagicMock()
    mock.returncode = status
    mock.stdout = stdout
    mock.stderr = stderr
    return mock


def fake_compose_binary_detector():
    """Return a fixed value of the compose command to avoid the subprocess call."""
    return ["foo", "bar"]
