import re

from core.version import get_version


def test_get_version():
    # Arrange
    # Major, minor, patch are numeric delimited by dots
    pattern = re.compile(r'\d+\.\d+\.\d+')

    # Act
    version = get_version()

    # Assert
    assert pattern.match(version) is not None
