import csv
import io
from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
from core.wrappers.server import Server
from core.wrappers.agent import Agent


class Kea(Agent):
    def __init__(self, compose: DockerCompose, service_name: str,
                 server_service: Server):
        super().__init__(compose, service_name, server_service)

    def read_lease_file(self, family: int):
        path = '/var/lib/kea/kea-leases%d.csv' % family
        stdout = self._read_file(path)
        return csv.DictReader(io.StringIO(stdout))

    def restart_stork_agent(self):
        self._restart_supervisor_service('stork-agent')

    def has_failed_TLS_handshake_log_entry(self):
        stdout, _ = self._compose.get_logs(self._service_name)
        return "HTTP_CONNECTION_HANDSHAKE_FAILED" in stdout
