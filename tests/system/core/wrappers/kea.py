import csv
import io

from core.wrappers.agent import Agent
from core.utils import wait_for_success, NoSuccessException


class Kea(Agent):
    """
    A wrapper for the docker-compose service containing Kea and Stork Agent.
    """
    prometheus_exporter_port = 9547

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
        path = f"/var/lib/kea/kea-leases{family}.csv"
        stdout = self._read_file(path)
        return csv.DictReader(io.StringIO(stdout))

    def has_failed_tls_handshake_log_entry(self):
        """Checks if any TLS handshake fail occurs."""
        stdout, _ = self._compose.logs(self._service_name)
        return "HTTP_CONNECTION_HANDSHAKE_FAILED" in stdout

    def has_encountered_unsupported_statistic(self):
        """Check if the Stork Agent Prometheus Exporter has encountered any
        unsupported statistics."""
        stdout, _ = self._compose.logs(self._service_name)
        return "Encountered unsupported stat" in stdout

    def wait_for_next_prometheus_metrics(self):
        """
        Block the execution until the Prometheus metrics are updated.
        In Kea exporter, in contrast to BIND 9 exporter, the metrics are not
        updated when the request is sent. The metrics are updated by the
        internal puller. This method waits for the metrics to be updated.
        """
        def get_metric_value(name):
            for metric in self.read_prometheus_metrics():
                if metric.name == name:
                    if len(metric.samples) == 0:
                        return None
                    return metric.samples[0].value
            return None

        metric = "storkagent_promkeaexporter_uptime_seconds"
        initial_value = get_metric_value(metric)

        @wait_for_success(wait_msg="Waiting to update Prometheus metrics...")
        def worker():
            value = get_metric_value(metric)
            if value == initial_value:
                raise NoSuccessException()
        worker()
