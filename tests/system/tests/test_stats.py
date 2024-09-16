from core.wrappers import Kea, Server, Perfdhcp


def test_get_kea_stats(
    server_service: Server, kea_service: Kea, perfdhcp_service: Perfdhcp
):
    """Check if collecting stats from various Kea versions works.
    DHCPv4 traffic is send to old Kea, then to new kea
    and then it is checked if Stork Server has collected
    stats about the traffic."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()

    perfdhcp_service.generate_ipv4_traffic(
        ip_address=kea_service.get_internal_ip_address("subnet_00", family=4),
        mac_prefix="00:00",
    )

    # ToDo: Add support for generation IPv6 traffic.
    # Unfortunately, there is a problem with starting the Kea DHCPv6 daemon.
    # It cannot bind the sockets. I didn't find any way to check if the
    # binding would be available. The Kea has implemented a solution to retry
    # the binding in kea#1716, but there is no possibility to check if the
    # binding has already finished successfully. I opened kea#2434 to add an
    # opportunity to inspect the binding status.
    #
    # perfdhcp_service.generate_ipv6_traffic(
    #     interface="eth1"
    # )

    server_service.wait_for_kea_statistics_pulling()
    data = server_service.overview()

    # 9 leases are initially store in the lease database
    assert int(data.dhcp4_stats.assigned_addresses) > 9
    assert data.subnets4.items is not None
    # ToDo: When we add support for IPv6 traffic generation
    # we will be able to test the number of assigned addresses
    # is greater than 9.
    assert int(data.dhcp6_stats.assigned_nas) == 9
    assert data.subnets6.items is not None

    # Check if Stork Agent handles all metrics returned by Kea.
    # The pool statistics are currently not supported.
    assert kea_service.has_encountered_unsupported_statistic()
