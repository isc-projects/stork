import csv
import io
from core.compose import DockerCompose
from core.wrappers.server import Server
from core.wrappers.agent import Agent


class Kea(Agent):
    """
    A wrapper for the docker-compose service containing Kea and Stork Agent.
    """

    def __init__(self, compose: DockerCompose, service_name: str,
                 server_service: Server):
        """
        A wrapper constructor.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service containing the Kea and
            Stork Agent.
        server_service : Server
            The wrapper for the Stork Server service where this agent is
            registered. If the registration was suppressed then it should be
            a None value.
        """
        super().__init__(compose, service_name, server_service)

    def read_lease_file(self, family: int):
        """
        Read a content of the lease file database.

        Parameters
        ----------
        family : int
            The IP family related to lease file. 4 or 6.

        Returns
        -------
        csv.DictReader
            The CSV reader ready to read the content.
        """
        path = '/var/lib/kea/kea-leases%d.csv' % family
        stdout = self._read_file(path)
        return csv.DictReader(io.StringIO(stdout))

    def has_failed_TLS_handshake_log_entry(self):
        """Checks if any TLS handshake fail occurs."""
        stdout, _ = self._compose.logs(self._service_name)
        return "HTTP_CONNECTION_HANDSHAKE_FAILED" in stdout
