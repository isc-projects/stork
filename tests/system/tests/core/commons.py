from unittest.mock import MagicMock


def subprocess_result_mock(status, stdout, stderr):
    mock = MagicMock()
    mock.returncode = status
    mock.stdout = stdout
    mock.stderr = stderr
    return mock


def fake_compose_detector():
    """Return a fixed value of the compose command and to avoid the
    subprocess call."""
    return ["docker", "compose"]
