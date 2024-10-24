import subprocess

from core.compose import DockerCompose
from core.utils import setup_logger


class Register:
    """
    A wrapper for the register command of the Stork agent service.
    This service container is available only when the service is executed.
    It isn't continuously running.
    """

    def __init__(self, compose: DockerCompose, service_name: str):
        """
        A wrapper constructor.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service that has the entrypoint set
            to the Stork agent "register" command.
        """
        self._compose = compose
        self._service_name = service_name
        self._logger = setup_logger("register")

    def register(self, server_token: str):
        """
        Performs the Stork agent registration with the provided server
        token (optional).
        """
        register_cmd = ["register", "--non-interactive"]
        if server_token is not None:
            register_cmd.append("--server-token")
            register_cmd.append(server_token)

        try:
            self._compose.run(self._service_name, *register_cmd)
        except subprocess.CalledProcessError as ex:
            # Log the stdout now because there is no possibility to capture
            # logs from the short-living containers at the end of test case.
            self._logger.info(ex.stdout)
            raise
