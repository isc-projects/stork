import re
import os.path

from core.utils import memoize
from core.constants import project_directory


@memoize
def get_version():
    """Returns the current Stork version from the GO file."""
    version_file = os.path.join(project_directory, "backend/version.go")
    pattern = re.compile(r'const Version = "(.*)"')
    with open(version_file, "rt", encoding="utf-8") as f:
        for line in f:
            match = pattern.match(line)
            if match is None:
                continue
            return match.group(1)
    # If we got here, the regex above failed.
    return "unknown"


def parse_version_info(version):
    """
    Parses the version string to version tuple. Returns a tuple of integers
    and a version suffix if it exists. Otherwise, None.

    Accepted values:
    - Major only: "1" -> (1,), None
    - Major, minor: "1.2" -> (1, 2), None
    - Major, minor, patch: "1.2.3" -> (1, 2, 3), None
    - Major, minor, patch, revision: "1.2.3.4" -> (1, 2, 3, 4), None
    - Version with suffix: "1.2.3-dev" -> (1, 2, 3), "dev"
    - Version with wildcard: "1.2.*" -> (1, 2), None
    - Version with suffix and wildcard: "1.2.*-dev" -> (1, 2), "dev"
    """
    parts = version.split(".")

    # The last part may contain a suffix after a dash.
    last_part = parts[-1]
    suffix = None
    if "-" in str(last_part):
        last_part, suffix = last_part.split("-", 1)
        parts[-1] = last_part

    # The last version part may be a wildcard. We skip it.
    if last_part == "*":
        parts = parts[:-1]

    # Convert the version parts to integers.
    return tuple(map(int, parts)), suffix


def get_version_info():
    """Returns the current Stork version tuple from the GO file."""
    version_info, _ = parse_version_info(get_version())
    return version_info
