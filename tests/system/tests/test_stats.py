from core.wrappers import Kea, Server, Perfdhcp


def test_get_kea_stats(server_service: Server, kea_service: Kea, perfdhcp_service: Perfdhcp):
    """Check if collecting stats from various Kea versions works.
       DHCPv4 traffic is send to old Kea, then to new kea
       and then it is checked if Stork Server has collected
       stats about the traffic."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()

    perfdhcp_service.generate_ipv4_traffic(
        ip_address=kea_service.get_internal_ip_address("subnet_00", family=4),
        mac_prefix="00:00"
    )

    # The perfdhcp fails when IPv6 address is used. Kea doesn't receive the
    # packets. There are 3 IPv6 interfaces, but the order is random. Kea
    # listens to 2 of them. I didn't find an easy way to recognize which
    # interface is assigned to a specific subnet. There is a temporary fix
    # that generates traffic on all interfaces.
    for interface in ("eth0", "eth1", "eth2"):
        perfdhcp_service.generate_ipv6_traffic(
            interface=interface
        )

    data = server_service.wait_for_update_overview()

    # 9 leases are initialy store in the lease database
    assert int(data['dhcp4_stats']['assignedAddresses']) > 9
    assert data['subnets4']['items'] is not None
    assert int(data['dhcp6_stats']['assignedNAs']) > 9
    assert data['subnets6']['items'] is not None
