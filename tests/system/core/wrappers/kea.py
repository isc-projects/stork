import csv
import io
from core.wrappers.agent import Agent


class Kea(Agent):
    """
    A wrapper for the docker-compose service containing Kea and Stork Agent.
    """

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

    def has_encountered_unsupported_statistic(self):
        """Check if the Stork Agent Prometheus Exporter has encountered any
        unsupported statistics."""
        stdout, _ = self._compose.logs(self._service_name)
        return "Encountered unsupported stat" in stdout
