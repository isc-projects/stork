from contextlib import contextmanager

from core.wrappers.compose import ComposeServiceWrapper


class Postgres(ComposeServiceWrapper):
    """A wrapper for the PostgreSQL docker-compose service."""

    @contextmanager
    def unavailable(self):
        """
        Context manager to temporarily pause the PostgreSQL service.

        The database service cannot detect that it was paused, so it should
        mimic the behavior of a service that is temporarily unavailable for
        example due to a connectivity issue.
        """
        self._compose.pause(self._service_name)
        yield
        self._compose.unpause(self._service_name)
        self._compose.wait_for_operational(self._service_name)

    @contextmanager
    def shutdown(self):
        """
        Context manager to temporarily stop the PostgreSQL service.

        The database service is completely stopped and cannot be accessed.
        On leaving the context manager, the service is started again. The
        database server gracefully shuts down and restarts.
        """
        self._compose.stop(self._service_name)
        yield
        self._compose.start(self._service_name)
        self._compose.wait_for_operational(self._service_name)

    def restart(self):
        """Restart the PostgreSQL service."""
        self._compose.restart(self._service_name)
        self._compose.wait_for_operational(self._service_name)
