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
    """Parses the version string to version tuple."""
    return tuple(int(x) for x in version.split("."))


def get_version_info():
    """Returns the current Stork version tuple from the GO file."""
    return parse_version_info(get_version())
