from core.wrappers.agent import Agent
from core.wrappers.server import Server
from core.compose import DockerCompose


class ExternalPackages(Agent, Server):
    """
    A wrapper for the docker-compose service containing Stork Server and
    Stork Agent installed from the external packages (CloudSmith).
    The image contains the current revision packages that can install on demand.
    """

    def __init__(self, compose: DockerCompose, service_name: str):
        """
        A wrapper constructor.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service containing the Bind9 and
            Stork Agent.
        """

        super().__init__(compose, service_name, self)
        self._agent_supervisor_service = self._get_supervisor_service("stork-agent")

    def _install_package(self, path):
        """Installs a given Debian package."""
        cmd = ["dpkg", "-i", path]
        self._compose.exec(self._service_name, cmd, capture_output=False)

    def update_agent_to_latest_version(self):
        """Installs the latest Stork Agent revision from the package."""
        package_path = "/app/dist/pkgs/isc-stork-agent.deb"
        self._install_package(package_path)
        self.restart_stork_agent()

    def update_server_to_latest_version(self):
        """Installs the latest Stork Server revision from the package."""
        package_path = "/app/dist/pkgs/isc-stork-server.deb"
        self._install_package(package_path)
        self.restart_stork_server()
