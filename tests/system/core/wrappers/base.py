from typing import Callable
from core.compose import DockerCompose
from core.log_parser import split_log_messages, GoLogEntry


class ComposeServiceWrapper:
    def __init__(self, compose: DockerCompose, service_name: str):
        self._compose = compose
        self._service_name = service_name

    def _restart_supervisor_service(self, name: str):
        cmd = ["supervisorctl", "restart", name]
        self._compose.exec(self._service_name, cmd)
        self._compose.wait_for_operational(self._service_name)

    def _read_file(self, path: str):
        cmd = ["cat", path]
        _, stdout, _ = self._compose.exec(
            self._service_name, cmd)
        return stdout

    def _hash_file(self, path: str):
        cmd = ["sha1sum", path]
        _, stdout, _ = self._compose.exec(self._service_name, cmd)
        return stdout.split()[0]

    def is_operational(self):
        return self._compose.is_operational(self._service_name)

    def search_for_logs(self, condition: Callable[[GoLogEntry], bool]):
        logs, _ = self._compose.logs(self._service_name)
        for entry in split_log_messages(logs):
            go_entry = entry.as_go()
            if condition(go_entry):
                yield go_entry
