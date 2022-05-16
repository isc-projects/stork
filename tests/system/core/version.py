import re
import os.path

from core.utils import memoize
from core.constants import project_directory


@memoize
def get_version():
    """Returns the current Stork version from the GO file."""
    version_file = os.path.join(project_directory, "backend/version.go")
    pattern = re.compile(r'const Version = "(.*)"')
    with open(version_file, "rt") as f:
        for line in f:
            match = pattern.match(line)
            if match is None:
                continue
            return match.group(1)
        