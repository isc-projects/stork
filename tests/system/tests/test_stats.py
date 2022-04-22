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
        ip_address=kea_service.get_ip_address("subnet_00"),
        mac_prefix="00:00"
    )
    perfdhcp_service.generate_ipv6_traffic(
        interface="eth2"
    )

    state, *_ = server_service.wait_for_next_machine_states()
