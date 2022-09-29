from core.fixtures import kea_parametrize
from core.wrappers import Server, Kea
import time

def test_reload_server_with_sighup(server_service: Server):
    # Remember current server's PID.
    pid_before = server_service.get_stork_server_pid()
    # Send SIGHUP.
    server_service.reload_stork_server()
    # The PID should not change and the process should not be restarted.
    pid_after = server_service.get_stork_server_pid()
    assert pid_before == pid_after
    # Capture the logs and make sure that the server has been reloaded.
    stdout, _ = server_service._compose.logs()
    assert "Reloading Stork Server after receiving SIGHUP signal" in stdout
