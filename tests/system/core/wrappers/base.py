from core.compose import DockerCompose


class ComposeServiceWrapper:
    """
    Base class for all continuously running docker-compose services.
    It wraps the docker-compose controller methods and low-level access to
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
        """Restart a specific supervisor service and wait to recover
        operational status."""
        cmd = ["supervisorctl", "restart", name]
        self._compose.exec(self._service_name, cmd)
        self._compose.wait_for_operational(self._service_name)

    def _reload_supervisor_service(self, name: str):
        cmd = ["supervisorctl", "signal", "HUP", name]
        self._compose.exec(self._service_name, cmd)
        self._compose.wait_for_operational(self._service_name)

    def _read_file(self, path: str):
        """Read a content of a given file from the container."""
        cmd = ["cat", path]
        _, stdout, _ = self._compose.exec(self._service_name, cmd)
        return stdout

    def _hash_file(self, path: str):
        """Calculates a hash of a given file from the container."""
        cmd = ["sha1sum", path]
        _, stdout, _ = self._compose.exec(self._service_name, cmd)
        return stdout.split()[0]

    def _download_file(self, source: str, target: str):
        """Downloads a file from the container to the host."""
        self._compose.copy_from_service(self._service_name, source, target)

    def is_operational(self):
        """Checks if the wrapped service is operational."""
        return self._compose.is_operational(self._service_name)

    def get_internal_ip_address(self, subnet_name: str, family: int):
        """
        Returns an internal Docker-network IP address from a given IP family
        assigned to the service in a given subnet.
        """
        return self._compose.get_service_ip_address(
            self._service_name, subnet_name, family=family
        )

    def _get_pid(self, process_name: str):
        """Returns a PID of the specfified process."""
        return self._compose.get_pid(self._service_name, process_name)
