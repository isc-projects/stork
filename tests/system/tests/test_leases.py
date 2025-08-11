"""System tests for the /leases API endpoint."""

# pylint: disable=import-error,no-name-in-module
# pylint: disable=unused-argument
import ipaddress
from core.wrappers import Server, Kea


def test_search_leases(kea_service: Kea, server_service: Server):
    """Test various lease search queries"""

    # Helper functions kept local to avoid extra imports
    def _is_ipv6(ip: str) -> bool:
        return ipaddress.ip_address(ip).version == 6

    def _is_ipv4(ip: str) -> bool:
        return ipaddress.ip_address(ip).version == 4

    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert len(state.apps) == 1
    version_raw = state.apps[0].version
    version = tuple(int(x) for x in version_raw.split("."))

    # Search by IPv4 address..
    data = server_service.list_leases("192.0.2.1")
    assert data.total == 1
    assert data.items[0].ip_address == "192.0.2.1"
    assert data.conflicts is None

    # Search by IPv6 address.
    data = server_service.list_leases("3001:db8:1:42::1")
    assert data.total == 1
    assert data.items[0].ip_address == "3001:db8:1:42::1"
    assert data.conflicts is None

    # Search by MAC.
    data = server_service.list_leases("00:01:02:03:04:02")
    assert data.total == 1
    assert data.items[0].ip_address == "192.0.2.2"
    assert data.conflicts is None

    # Search by client id and DUID.
    data = server_service.list_leases("01:02:03:04")
    assert data.total == 2
    assert data.items[0].ip_address == "192.0.2.4"
    assert data.items[1].ip_address == "3001:db8:1:42::4"
    assert data.conflicts is None

    # Search by hostname.
    data = server_service.list_leases("host-6.example.org")
    assert data.total == 2
    assert data.items[0].ip_address == "192.0.2.6"
    assert data.items[1].ip_address == "3001:db8:1:42::6"
    assert data.conflicts is None

    # Search declined leases.
    data = server_service.list_leases("state:declined")

    # The Kea prior 2.3.8 used a single "0" as an empty DUID.
    # The lease file is generated before running the container, so we don't
    # know the Kea version beforehand and we cannot use the proper value.
    # All generated declined leases have the "00:00:00" DUID, so the old Kea
    # DHCP daemon doesn't recognize them as declined. I didn't find a
    # workaround without completely rewriting the lease generation.
    assert data.total == 20 if version >= (2, 3, 8) else 10

    for lease in data.items:
        # Declined leases should lack identifiers.
        assert lease.hw_address is None
        assert lease.client_id is None
        assert lease.ip_address is not None
        if _is_ipv6(lease.ip_address):
            assert lease.duid == "00:00:00"
        else:
            assert lease.duid is None
        # The state is declined.
        assert lease.state == 1

    # Sanity checks of leases.
    assert data.items[0].ip_address == "192.0.2.1"
    if version >= (2, 3, 8):
        assert data.items[10].ip_address == "3001:db8:1:42::1"
    assert data.conflicts is None

    # Blank search text should return none leases
    data = server_service.list_leases()
    assert data.items is None


def test_get_host_leases(kea_service: Kea, server_service: Server):
    """Test getting leases for a host."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()

    # Find the host reservation for IPv4 address.
    hosts = server_service.list_hosts("192.0.2.2")
    host = hosts.items[0]

    # Find the leases for the host.
    leases = server_service.list_leases(host_id=host.id)
    assert leases.total == 1
    assert leases.items[0].ip_address == "192.0.2.2"
    assert leases.conflicts is None

    # Find the host reservation for IPv6 address.
    hosts = server_service.list_hosts("3001:db8:1:42::2")
    host = hosts.items[0]

    # Find leases for the IPv6 host reservation.
    leases = server_service.list_leases(host_id=host.id)
    assert leases.total == 1
    assert leases.items[0].ip_address == "3001:db8:1:42::2"

    # The lease was assigned to a different client. There should be
    # a conflict returned.
    assert len(leases.conflicts) == 1
    assert leases.items[0].id == leases.conflicts[0]

    # Find leases for non-existing host id.
    leases = server_service.list_leases(host_id=1000000)
    assert leases.items is None
