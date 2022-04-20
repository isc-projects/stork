import csv
import io
from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
from core.wrappers.server import Server
from core.utils import wait_for_success, NoSuccessException


class Kea(ComposeServiceWrapper):
    def __init__(self, compose: DockerCompose, service_name: str,
                 server_service: Server):
        super().__init__(compose, service_name)
        self._server_service = server_service

    @property
    def server(self):
        return self._server_service

    def is_registered(self):
        if self._server_service is None:
            return False
        stdout, _ = self._compose.get_logs()
        return "machine registered" in stdout

    def read_lease_file(self, family: int):
        path = '/var/lib/kea/kea-leases%d.csv' % family
        cmd = ["cat", path]
        _, stdout, _ = self._compose.exec_in_container(
            self._service_name, cmd)

        return csv.DictReader(io.StringIO(stdout))

    @wait_for_success()
    def wait_for_registration(self):
        if not self.is_registered():
            raise NoSuccessException()
