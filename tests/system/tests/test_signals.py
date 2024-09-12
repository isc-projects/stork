from core.fixtures import kea_parametrize, server_parametrize
from core.wrappers import Server, Kea
from core.compose_factory import create_docker_compose


@server_parametrize("server")
def test_reload_server_with_sighup(server_service: Server):
    # Remember current server's PID.
    pid_before = server_service.get_stork_server_pid()
    # Send SIGHUP.
    server_service.reload_stork_server()
    # The PID should not change and the process should not be restarted.
    pid_after = server_service.get_stork_server_pid()
    assert pid_before == pid_after
    # Capture the logs and make sure that the server has been reloaded.
    compose = create_docker_compose()
    stdout, _ = compose.logs()
    assert "Reloading Stork Server after receiving SIGHUP signal" in stdout


@kea_parametrize("agent-kea")
@server_parametrize("server")
def test_reload_agent_with_sighup(server_service: Server, kea_service: Kea):
    # Remember current agent's PID.
    pid_before = kea_service.get_stork_agent_pid()
    # Send SIGHUP.
    kea_service.reload_stork_agent()
    # The PID should not change and the process should not be restarted.
    pid_after = kea_service.get_stork_agent_pid()
    assert pid_before == pid_after
    # Capture the logs and make sure that the agent has been reloaded.
    compose = create_docker_compose()
    stdout, _ = compose.logs()
    assert "Reloading Stork Agent after receiving SIGHUP signal" in stdout
