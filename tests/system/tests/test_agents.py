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
    state, *_ = server_service.wait_for_next_machine_states()

    assert state['apps'] is not None
    assert len(state['apps']) == 1
    assert state['apps'][0]['version'] == "2.0.2"
    assert len(state['apps'][0]['accessPoints']) == 1
    assert state['apps'][0]['accessPoints'][0]['address'] == '127.0.0.1'

    server_service.wait_for_adding_subnets(daemon_name="dhcp4")

    subnets = server_service.list_subnets(family=4)
    assert subnets["total"] == 6912


def test_search_leases(kea_service: Kea, server_service: Server):
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()

    # Search by IPv4 address..
    data = server_service.list_leases('192.0.2.1')
    assert data['total'] == 1
    assert data['items'][0]['ipAddress'] == '192.0.2.1'
    assert data['conflicts'] is None

    # Search by IPv6 address.
    data = server_service.list_leases('3001:db8:1::1')
    assert data['total'] == 1
    assert data['items'][0]['ipAddress'] == '3001:db8:1::1'
    assert data['conflicts'] is None

    # Search by MAC.
    data = server_service.list_leases('00:01:02:03:04:02')
    assert data['total'] == 1
    assert data['items'][0]['ipAddress'] == '192.0.2.2'
    assert data['conflicts'] is None

    # Search by client id and DUID.
    data = server_service.list_leases('01:02:03:04')
    assert data['total'] == 2
    assert data['items'][0]['ipAddress'] == '192.0.2.4'
    assert data['items'][1]['ipAddress'] == '3001:db8:1::4'
    assert data['conflicts'] is None

    # Search by hostname.
    data = server_service.list_leases('host-6.example.org')
    assert data['total'] == 2
    assert data['items'][0]['ipAddress'] == '192.0.2.6'
    assert data['items'][1]['ipAddress'] == '3001:db8:1::6'
    assert data['conflicts'] is None

    # Search declined leases.
    data = server_service.list_leases('state:declined')
    assert data['total'] == 20
    for lease in data['items']:
        # Declined leases should lack identifiers.
        assert 'hwaddr' not in lease
        assert 'clientId' not in lease
        assert 'duid' not in lease
        # The state is declined.
        assert lease['state'] == 1
        # An address should be set.
        assert 'ipAddress' in lease
    # Sanity check addresses returned.
    assert data['items'][0]['ipAddress'] == '192.0.2.1'
    assert data['items'][10]['ipAddress'] == '3001:db8:1::1'
    assert data['conflicts'] is None

    # Blank search text should return none leases
    data = server_service.list_leases()
    assert data['items'] is None
