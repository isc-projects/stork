class ServiceState:
    """
    Container for the current service state details.
    """

    def __init__(self, status, exit_code, health, health_details):
        """
        Constructs the service state.

        Parameters
        ----------
        status : str
            Docker container status.
        exit_code : int
            Exit code or 0 for running.
        health : str
            Docker health status
        health_details : str
            Health details serialized to JSON.
        """
        self._status = status
        self._exit_code = exit_code
        self._health = health
        self._health_details = health_details

    def is_running(self):
        """True if container has running status."""
        return self._status == "running"

    def has_healthcheck(self):
        """True if the health status is available."""
        return self._health is not None

    def is_healthy(self):
        """True if container is healthy or healthcheck is not defined."""
        if not self.has_healthcheck():
            return True
        return self._health == "healthy"

    def is_unhealthy(self):
        """True if container is unhealthy."""
        return self.has_healthcheck() and self._health == "unhealthy"

    def is_exited(self):
        """True if container has exited status."""
        return self._status == "exited"

    def is_starting(self):
        """True if container has starting status."""
        return self._status == "starting"

    def is_operational(self):
        """Complex status. True if container is running and healthy
        (if available)."""
        return self.is_running() and self.is_healthy()

    def __str__(self):
        """Converts state to string."""
        if self.is_exited():
            return f"exited (code: {self._exit_code})"
        if self.is_unhealthy():
            return f"{self._status} ({self._health})\n{self._health_details}"
        return f"{self._status} ({self._health})"
