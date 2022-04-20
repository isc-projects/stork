
from datetime import datetime
import shlex


class GoLogEntry:
    def __init__(self, raw: str):
        severity_with_timestamp, content = raw.split(" ", 1)

        content = content.lstrip()
        location, content = content.split(" ", 1)
        content = content.lstrip()

        self._severity_with_timestamp = severity_with_timestamp
        self._location = location
        self._content = content

    @property
    def severity(self):
        severity, _ = self._severity_with_timestamp.split("[")
        return severity

    @property
    def timestamp(self):
        _, timestamp = self._severity_with_timestamp.split("[")
        timestamp = timestamp.rstrip("]")
        return datetime.strptime(timestamp, "%Y-%m-%d %H:%M:%S")

    @property
    def location(self):
        return self._location

    @property
    def location_file(self):
        return self._location.rsplit(":")[0]

    @property
    def location_line(self):
        return int(self._location.rsplit(":")[1])

    @property
    def message(self):
        message, _ = self._content.split("    ")
        return message

    @property
    def arguments(self):
        _, arguments_str = self._content.split("    ")
        arguments_str = arguments_str.lstrip()
        arguments_dict = {}
        for argument in shlex.split(arguments_str):
            key, value = argument.split("=", 1)
            arguments_dict[key] = value
        return arguments_dict


class LogEntry:
    @staticmethod
    def _split_docker_compose_log_entry(raw: str, with_timestamp=True):
        service_name, rest = raw.split("|", 1)
        service_name = service_name.strip()
        rest = rest.strip()

        if not with_timestamp:
            service_name, rest

        timestamp, rest = rest.split(" ", 1)
        return service_name, timestamp, rest

    def __init__(self, raw):
        service_name, timestamp, content = LogEntry._split_docker_compose_log_entry(
            raw)
        self._service_name = service_name
        self._timestamp = timestamp
        self._content = content

    @property
    def service_name(self):
        return self._service_name

    @property
    def timestamp(self):
        return datetime.fromisoformat(self._timestamp.rstrip("Z"))

    def as_go(self):
        return GoLogEntry(self._content)


def split_log_messages(stdout: str):
    for line in stdout.split("\n"):
        line = line.rstrip()
        entry = LogEntry(line)
        yield entry
