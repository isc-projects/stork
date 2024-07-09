import os.path
import urllib3

from core.compose import DockerCompose
from core.wrappers.compose import ComposeServiceWrapper
from core.wrappers.server import Server
from core.utils import memoize, wait_for_success, NoSuccessException
from core.prometheus_parser import text_fd_to_metric_families


class Agent(ComposeServiceWrapper):
    """A wrapper for the Stork Agent docker-compose service."""

    _cert_paths = [
        "/var/lib/stork-agent/certs/key.pem",
        "/var/lib/stork-agent/certs/cert.pem",
        "/var/lib/stork-agent/certs/ca.pem",
        "/var/lib/stork-agent/tokens/agent-token.txt",
    ]

    prometheus_exporter_port = 0  # Unknown port

    def __init__(
        self, compose: DockerCompose, service_name: str, server_service: Server
    ):
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
        self._agent_supervisor_service = self._get_supervisor_service("stork-agent")

    @memoize
    def _get_metrics_endpoint(self, internal_port: int):
        """Returns URL of the agent metrics endpoint."""
        mapped = self._compose.port(self._service_name, internal_port)
        url = f"http://{mapped[0]}:{mapped[1]}/metrics"
        return url

    @memoize
    def get_stork_control_endpoint(self):
        """Returns the host and port of the stork-agent control endpoint."""
        internal_port = 8080
        mapped = self._compose.port(self._service_name, internal_port)
        return mapped

    @property
    def server(self):
        """Returns a Server wrapper where this agent is registered. If the
        registration was suppressed then it returns None."""
        return self._server_service

    def hash_cert_files(self):
        """Calculates the hashes of the TLS credentials used by the agent."""
        hashes = {}
        for cert_path in Agent._cert_paths:
            hash_ = self._hash_file(cert_path)
            hashes[cert_path] = hash_
        return hashes

    def download_cert_files(self, target_dir: str):
        """Downloads the TLS credentials used by the agent to the host."""
        downloaded = {}
        for cert_path in Agent._cert_paths:
            filename = os.path.basename(cert_path)
            target = os.path.join(target_dir, filename)
            self._download_file(cert_path, target)
            downloaded[os.path.splitext(filename)[0]] = target
        return downloaded

    def read_prometheus_metrics(self):
        """
        Reads the Prometheus metrics collected by the Stork Agent.
        Returns a dictionary where the key is the metric name and the value is
        the metric object.
        """
        if self.prometheus_exporter_port == 0:
            raise ValueError("The Prometheus exporter port is not known.")

        url = self._get_metrics_endpoint(self.prometheus_exporter_port)
        http = urllib3.PoolManager()
        resp = http.request("GET", url)
        data = resp.data.decode("utf-8")
        data = data.split("\n")

        metrics_list = text_fd_to_metric_families(data)
        metrics_dict = {metric.name: metric for metric in metrics_list}
        return metrics_dict

    def restart_stork_agent(self):
        """
        Restarts the Stork Agent and waits to recover an operational status.
        """
        self._agent_supervisor_service.restart()

    def reload_stork_agent(self):
        """Sends SIGHUP to the stork-agent."""
        self._agent_supervisor_service.reload()

    def is_registered(self):
        """True if an agent was successful registered. Otherwise False."""
        if self._server_service is None:
            return False
        # ToDo: Using logs is a little dangerous. They can contain a bloat data.
        stdout, _ = self._compose.logs()
        return "machine registered" in stdout.lower()

    @wait_for_success(wait_msg="Waiting to be registered...")
    def wait_for_registration(self):
        """Block the execution until registration passes."""
        if not self.is_registered():
            raise NoSuccessException()

    def get_stork_agent_pid(self):
        """Returns PID of the stork-agent process."""
        return self._agent_supervisor_service.get_pid()
