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

    def has_number_overflow_log_entry(self):
        """Check if any number overflow error from Kea is present in the logs."""
        stdout, _ = self._compose.logs(self._service_name)
        return (
            "non-success response result from Kea: 1, text: internal server "
            "error: unable to parse server's answer to the forwarded message: "
            "Number overflow:"
        ) in stdout

    def has_encountered_machine_registration_disabled(self):
        """Check if the Stork Agent has encountered an error indicating that
        new machine registration is administratively disabled."""
        stdout, _ = self._compose.logs(self._service_name)
        return "Machine registration is administratively disabled" in stdout

    def get_version(self):
        """Returns the Kea version as a tuple."""
        stdout: str
        _, stdout, _ = self._compose.exec(self._service_name, ["kea-ctrl-agent", "-v"])
        return tuple(int(i) for i in stdout.strip().split("."))

    def wait_for_detect_kea_daemons(self, expected_daemons: int = 2):
        """
        Wait for the Stork Agent to detect the Kea daemons.

        It accepts the number of expected daemons and waits until the
        Stork agent detects them.
        """

        @wait_for_success(wait_msg="Waiting for the Kea daemons to be detected...")
        def worker():
            metrics = self.wait_for_next_prometheus_metrics()

            # Wait for daemons.
            monitored_daemons = Kea._get_metric_int_value(
                metrics, "storkagent_monitor_monitored_kea_daemons_total", 0
            )
            if monitored_daemons < expected_daemons:
                raise NoSuccessException()

        worker()

    def wait_for_next_prometheus_metrics(self):
        """
        Block the execution until the Prometheus metrics are updated.
        In Kea exporter, in contrast to BIND 9 exporter, the metrics are not
        updated when the request is sent. The metrics are updated by the
        internal puller. This method waits for the metrics to be updated.
        """
        uptime_metric_name = "storkagent_promkeaexporter_uptime_seconds"
        initial_uptime = Kea._get_metric_value(
            self.read_prometheus_metrics(), uptime_metric_name
        )

        @wait_for_success(wait_msg="Waiting to update Prometheus metrics...")
        def worker():
            metrics = self.read_prometheus_metrics()
            uptime = Kea._get_metric_value(metrics, uptime_metric_name)
            if uptime == initial_uptime:
                raise NoSuccessException()
            return metrics

        return worker()

    @staticmethod
    def _get_metric_value(metrics, name, default_=None):
        metric = metrics.get(name)
        if metric is None:
            return default_
        if len(metric.samples) == 0:
            return default_
        return metric.samples[0].value

    @staticmethod
    def _get_metric_int_value(metrics, name, default_=None):
        value = Kea._get_metric_value(metrics, name)
        if value is None:
            return default_
        return int(round(value))
