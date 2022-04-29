from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
from core.wrappers.server import Server
from core.utils import wait_for_success, NoSuccessException


class Agent(ComposeServiceWrapper):
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
        stdout, _ = self._compose.logs()
        return "machine registered" in stdout

    @wait_for_success()
    def wait_for_registration(self):
        if not self.is_registered():
            raise NoSuccessException()

    def hash_cert_files(self):
        cert_paths = [
            '/var/lib/stork-agent/certs/key.pem',
            '/var/lib/stork-agent/certs/cert.pem',
            '/var/lib/stork-agent/certs/ca.pem',
            '/var/lib/stork-agent/tokens/agent-token.txt'
        ]

        hashes = {}
        for cert_path in cert_paths:
            hash_ = self._hash_file(cert_path)
            hashes[cert_path] = hash_
        return hashes
