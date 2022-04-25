
from datetime import datetime, timezone
import shlex


class GoLogEntry:
    def __init__(self, raw: str):
        severity_with_timestamp, content = raw.split("]", 1)

        content = content.lstrip()
        location, content = content.split(" ", 1)
        content = content.lstrip()

        self._severity_with_timestamp = severity_with_timestamp
        self._location = location
        self._content = content

    @staticmethod
    def _remove_colors(raw: str):
        characters = []
        skip = 0
        for char in raw:
            if skip > 0:
                skip -= 1
                continue
            if char == "\x1b":
                skip = 4
                continue
            characters.append(char)
        return "".join(characters)

    @property
    def severity(self):
        severity, _ = self._severity_with_timestamp.rsplit("[", 1)
        return GoLogEntry._remove_colors(severity)

    @property
    def timestamp(self):
        _, timestamp = self._severity_with_timestamp.rsplit("[", 1)
        dt = datetime.strptime(timestamp, "%Y-%m-%d %H:%M:%S")
        return dt.replace(tzinfo=timezone.utc)

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
        message, *_ = self._content.split("    ", 1)
        return message

    @property
    def arguments(self):
        _, arguments_str = self._content.split("    ", 1)
        arguments_str = self._remove_colors(arguments_str)
        arguments_str = arguments_str.lstrip()
        arguments_dict = {}
        for argument in shlex.split(arguments_str):
            key, value = argument.split("=", 1)
            arguments_dict[key] = value
        return arguments_dict


class KeaLogEntry:
    def __init__(self, raw: str):
        severity, rest = raw.split(maxsplit=1)
        rest = rest.lstrip()
        id_, message = rest.split(maxsplit=1)
        message = rest.strip()

        self._severity = severity
        self._message = message
        self._id = id_

    @property
    def severity(self):
        return self._severity

    @property
    def message_id(self):
        return self._id

    @property
    def message(self):
        return self._message


class LogEntry:
    @staticmethod
    def _split_docker_compose_log_entry(raw: str, with_timestamp=True):
        part = raw.split("|", 1)
        if len(part) == 1:
            return None, None, part
        service_name, rest = part
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
        dt = datetime.fromisoformat(self._timestamp[:26])
        return dt.replace(tzinfo=timezone.utc)

    def is_service_log_entry(self):
        return self._service_name is not None and self._timestamp is not None \
            and self._content is not None

    def is_service(self, service_name):
        if self._service_name is None:
            return False
        name, _ = self._service_name.rsplit("_", 1)
        return name == service_name

    def as_go(self):
        return GoLogEntry(self._content)

    def as_go_safe(self):
        try:
            return self.as_go()
        except:
            return None

    def as_kea(self):
        return KeaLogEntry(self._content)


def split_log_messages(stdout: str):
    for line in stdout.split("\n"):
        line = line.rstrip()
        entry = LogEntry(line)
        yield entry
