from os import stat
from core.wrappers import Kea, Server
from core.fixtures import kea_parametrize


def test_agent_reregistration_after_restart(server_service: Server, kea_service: Kea):
    """Check if after restart the agent isn't re-register.
       It should use the same agent token and certs as before restart."""
    server_service.log_in_as_admin()
    machine_before = server_service.authorize_all_machines()['items'][0]
    hashes_before = kea_service.hash_cert_files()

    kea_service.restart_stork_agent()

    machine_after = server_service.list_machines()['items'][0]
    hashes_after = kea_service.hash_cert_files()

    assert machine_before["agentToken"] == machine_after['agentToken']
    assert hashes_before == hashes_after


@kea_parametrize("agent-kea6")
def test_agent_over_ipv6(server_service: Server, kea_service: Kea):
    server_service.log_in_as_admin()
    machine = server_service.authorize_all_machines()['items'][0]

    assert ":" in machine['address']

    state = server_service.wait_for_next_machine_state(machine['id'])

    assert len(state["apps"]) > 0
