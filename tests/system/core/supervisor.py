from typing import Callable, Tuple, Sequence

from core.utils import NoSuccessException, wait_for_success


class SupervisorService:
    """A wrapper for the supervisor service."""

    def __init__(
        self, exec_: Callable[[Sequence[str]], Tuple[int, str, str]], service_name: str
    ):
        """
        A constructor of the class.

        Parameters
        ----------
        exec_ : Callable[[Sequence[str]], Tuple[str, str, int]]
            A function to execute a command in the container shell.
        service_name : str
            The name of the supervisor service.

        Notes
        -----
        Currently, all methods interact with the supervisor by calling the
        supervisorctl command. It may be replaced with the SupervisorD RPC
        interface in the future.
        """
        self._service_name = service_name
        self._exec = exec_

    def get_pid(self):
        """Returns a PID of a specific supervisor service."""
        cmd = ["supervisorctl", "pid", self._service_name]
        _, stdout, _ = self._exec(cmd)
        return int(stdout)

    def is_operational(self):
        """Checks if a specific supervisor service is operational."""
        cmd = ["supervisorctl", "status", self._service_name]
        # See: https://supervisord.org/subprocess.html#process-states for
        # the list of process states.
        code, stdout, _ = self._exec(cmd)
        # If any process is in the STARTING or BACKOFF state, it means they are
        # not operational. BACKOFF means that the process failed to start
        # and is waiting for the retry.
        if "BACKOFF" in stdout or "STARTING" in stdout:
            return False

        return code == 0

    def restart(self, wait_for_operational: bool = True):
        """Restart a specific supervisor service and wait to recover
        operational status."""
        cmd = ["supervisorctl", "restart", self._service_name]
        self._exec(cmd)
        if wait_for_operational:
            self._wait_for_operational()

    def _send_signal(self, signal: str):
        """Send a specific signal to a supervisor service."""
        cmd = ["supervisorctl", "signal", signal, self._service_name]
        self._exec(cmd)

    def reload(self):
        """Reload a specific supervisor service and wait to recover
        operational status."""
        self._send_signal("HUP")
        self._wait_for_operational()

    def interrupt(self):
        """Interrupt a specific supervisor service and wait to stop the
        service."""
        self._send_signal("INT")
        self._wait_for_non_operational()

    @wait_for_success(wait_msg="Waiting to be supervisor service operational...")
    def _wait_for_operational(self):
        """Block the execution until the service is operational."""
        if not self.is_operational():
            raise NoSuccessException()

    @wait_for_success(wait_msg="Waiting to be supervisor service non-operational...")
    def _wait_for_non_operational(self):
        """Block the execution until the service is non-operational."""
        if self.is_operational():
            raise NoSuccessException()
