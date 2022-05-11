from core.compose import DockerCompose
from core.wrappers.base import ComposeServiceWrapper
from core.wrappers.server import Server
from core.utils import wait_for_success, NoSuccessException


class Agent(ComposeServiceWrapper):
    """A wrapper for the Stork Agent docker-compose service."""

    def __init__(self, compose: DockerCompose, service_name: str,
                 server_service: Server):
        """
        A wrapper constructor.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service containing the Stork Agent.
        server_service : Server
            The wrapper for the Stork Server service where this agent is
            registered. If the registration was suppressed then it should be
            a None value.
        """
        super().__init__(compose, service_name)
        self._server_service = server_service

    @property
    def server(self):
        """Returns a Server wrapper where this agent is registered. If the
        registration was suppresses then it returns None."""
        return self._server_service

    def hash_cert_files(self):
        """Calculates the hashes of the TLS credentials used by the agent."""
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
