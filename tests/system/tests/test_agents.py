import time
from core.fixtures import kea_parametrize
from core.wrappers import Server, Kea


@kea_parametrize("agent-kea-many-subnets")
def test_add_kea_with_many_subnets(server_service: Server, kea_service: Kea):
    """Check if Stork agent and server will handle Kea instance with huge amount of subnets."""
    kea_service.wait_for_registration()
    server_service.log_in_as_admin()
    machines = server_service.authorize_all_machines()
    assert len(machines["items"]) == 1
    machine = machines["items"][0]
    state = server_service.wait_for_next_machine_state(machine["id"])

    assert state['apps'] is not None
    assert len(state['apps']) == 1
    assert state['apps'][0]['version'] == "2.0.2"
    assert len(state['apps'][0]['accessPoints']) == 1
    assert state['apps'][0]['accessPoints'][0]['address'] == '127.0.0.1'

    server_service.wait_for_adding_subnets(daemon_name="dhcp4")

    subnets = server_service.list_subnets(family=4)
    assert subnets["total"] == 6912
