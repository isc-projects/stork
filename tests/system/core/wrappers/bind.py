from core.compose import DockerCompose
from core.wrappers.server import Server
from core.wrappers.agent import Agent


class Bind9(Agent):
    """
    A wrapper for the docker-compose service containing Bind9 and Stork Agent.
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
            The name of the docker-compose service containing the Bind9 and
            Stork Agent.
        server_service : Server
            The wrapper for the Stork Server service where this agent is
            registered. If the registration was suppressed then it should be
            a None value.
        """
        super().__init__(compose, service_name, server_service)
