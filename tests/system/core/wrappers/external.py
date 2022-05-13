from core.wrappers.agent import Agent
from core.wrappers.server import Server
from core.compose import DockerCompose


class ExternalPackages(Agent, Server):
    """
    A wrapper for the docker-compose service containing Stork Server and
    Stork Agent installed from the external packages (CloudSmith).
    """

    def __init__(self, compose: DockerCompose, service_name: str):
        Server.__init__(self, compose, service_name)
        Agent.__init__(self, compose, service_name, self)

    def restart_stork_server(self):
        """
        Restarts the Stork Server and waits to recover an operational status.
        """
        self._restart_supervisor_service('stork-server')