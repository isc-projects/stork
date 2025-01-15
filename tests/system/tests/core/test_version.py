import re
from unittest.mock import patch

import pytest

import core.version


def test_get_version():
    # Arrange
    # Major, minor, patch are numeric delimited by dots
    pattern = re.compile(r"\d+\.\d+\.\d+")

    # Act
    version = core.version.get_version()

    # Assert
    assert pattern.match(version) is not None


@patch("core.version.get_version", return_value="1.2.3")
def test_get_version_info(_):
    info = core.version.get_version_info()
    assert info == (1, 2, 3)


def test_parse_version_info_positive():
    test_cases = (
        # Input and expected output.
        ("1", ((1,), None)),  # Major only.
        ("1.2", ((1, 2), None)),  # Major, minor.
        ("1.2.3", ((1, 2, 3), None)),  # Major, minor, patch.
        ("1.2.3.4", ((1, 2, 3, 4), None)),  # Major, minor, patch, revision.
        ("1.2.3-dev", ((1, 2, 3), "dev")),  # Version with suffix.
        ("1.2.*", ((1, 2), None)),  # Version with wildcard.
        ("1.2.*-dev", ((1, 2), "dev")),  # Version with suffix and wildcard.
        ("1.2.3-dev-rc2", ((1, 2, 3), "dev-rc2")),  # Suffix with many dashes.
    )

    for version, (expected_info, expected_suffix) in test_cases:
        info, suffix = core.version.parse_version_info(version)
        assert info == expected_info
        assert suffix == expected_suffix


def test_parse_version_info_negative():
    test_cases = (
        "",  # Empty string.
        "foobar",  # Non-numeric string.
        "1.2.",  # Missing patch.
        "1.2.foo",  # Non-numeric patch.
        "1..2.3",  # Double dots.
    )

    for version in test_cases:
        with pytest.raises(Exception):
            core.version.parse_version_info(version)
