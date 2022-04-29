from core.compose import DockerCompose


class ComposeServiceWrapper:
    """
    Base class for all continuously running docker-compose services.
    It wraps the docker-compose controler methods and low-level access to
    the container filesystem.
    """

    def __init__(self, compose: DockerCompose, service_name: str):
        """
        A constructor of the class.

        Parameters
        ----------
        compose : DockerCompose
            The compose controller object
        service_name : str
            The name of the docker-compose service.
        """
        self._compose = compose
        self._service_name = service_name

    def _restart_supervisor_service(self, name: str):
        """Restart a specific supervisor service and waits to recover
        operational status."""
        cmd = ["supervisorctl", "restart", name]
        self._compose.exec(self._service_name, cmd)
        self._compose.wait_for_operational(self._service_name)

    def _read_file(self, path: str):
        """Read a content of a given file from the container."""
        cmd = ["cat", path]
        _, stdout, _ = self._compose.exec(
            self._service_name, cmd)
        return stdout

    def _hash_file(self, path: str):
        """Calculates a hash of a given file from the container."""
        cmd = ["sha1sum", path]
        _, stdout, _ = self._compose.exec(self._service_name, cmd)
        return stdout.split()[0]

    def is_operational(self):
        """Checks if the wrapped service is operational."""
        return self._compose.is_operational(self._service_name)

    def get_ip_address(self, subnet_name: str):
        """Returns an IP address assigned to the service in a given subnet"""
        return self._compose.get_service_ip_address(self._service_name,
                                                    subnet_name)
