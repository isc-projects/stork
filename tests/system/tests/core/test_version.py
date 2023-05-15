import re
from unittest.mock import patch, Mock

import pytest

import core.version


def test_get_version():
    # Arrange
    # Major, minor, patch are numeric delimited by dots
    pattern = re.compile(r'\d+\.\d+\.\d+')

    # Act
    version = core.version.get_version()

    # Assert
    assert pattern.match(version) is not None


@patch("core.version.get_version", return_value="1.2.3")
def test_get_version_info(mock):
    info = core.version.get_version_info()
    assert info == (1, 2, 3)


def test_parse_version_info():
    info = core.version.parse_version_info("11.22.33")
    assert info == (11, 22, 33)
