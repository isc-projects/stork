from unittest.mock import MagicMock


def subprocess_result_mock(status, stdout, stderr):
    mock = MagicMock()
    mock.returncode = status
    mock.stdout = stdout
    mock.stderr = stderr
    return mock
