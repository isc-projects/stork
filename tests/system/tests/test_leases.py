"""System tests for the /leases API endpoint."""

import ipaddress

from core.wrappers import Server, Kea


def helper_declined_leases(version: (int, int, int), server_service: Server):
    """Test the declined leases search query specifically.

    This is a helper function for test_search_leases because that function got
    too long and the linter complained about it.
    """
    data = server_service.list_leases("state:declined")

    # The Kea prior 2.3.8 used a single "0" as an empty DUID.
    #
    # The lease file is generated before running the container, so we don't
    # know the Kea version beforehand and we cannot use the proper value.
    # All generated declined leases have the "00:00:00" DUID, so the old Kea
    # DHCP daemon doesn't recognize them as declined. I didn't find a
    # workaround without completely rewriting the lease generation.
    #
    # Kea versions 3.1.1 through 3.1.4 do not support the state:declined query
    # at all. The method described above was identified as an input validation
    # bug and fixed in 3.1.1, and a replacement API (leases[46]-get-by-state)
    # was not added until 3.1.5.
    if (3, 1, 0) < version < (3, 1, 5):
        assert data.total is None
        assert len(data.erred_apps) == 2
    elif version >= (2, 3, 8):
        assert data.total == 20
    else:
        assert data.total == 10

    if data.items is not None:
        for lease in data.items:
            # Declined leases should lack identifiers.
            assert lease.hw_address is None
            assert lease.client_id is None
            assert lease.ip_address is not None
            if ipaddress.ip_address(lease.ip_address).version == 6:
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


def test_search_leases(kea_service: Kea, server_service: Server):
    """Test various lease search queries"""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    daemons = [d for a in state.apps for d in a.details.daemons]
    assert len(daemons) == 3
    version_raw = daemons[0].version
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
    helper_declined_leases(version, server_service)

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
