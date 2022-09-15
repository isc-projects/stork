from core.wrappers import Server, Kea, Perfdhcp
from core.fixtures import kea_parametrize


@kea_parametrize("agent-kea-premium-host-database")
def test_get_host_reservation_from_host_db(kea_service: Kea, server_service: Server):
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_host_reservation_pulling()

    hosts = server_service.list_hosts('192.0.2.42')
    assert hosts is not None
    assert len(hosts.items) == 1
    host = hosts.items[0]
    local_hosts = host["localHosts"]
    len(local_hosts) == 1
    local_host = local_hosts[0]
    assert local_host["dataSource"] == "api"


@kea_parametrize("agent-kea-premium-radius")
def test_get_host_reservations_from_radius(kea_service: Kea, server_service: Server, perfdhcp_service: Perfdhcp):
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()

    # The RADIUS backend is initialized on the first call of the "selectSubnet"
    # callout. Perfdhcp generates the network traffic that triggers this call.
    perfdhcp_service.generate_ipv4_traffic(
        ip_address=kea_service.get_internal_ip_address("subnet_00", family=4),
        mac_prefix="00:00"
    )

    # Waits for send the "reservation-get-page" command to Kea.
    server_service.wait_for_host_reservation_pulling()

    # There should be no communication failed events.
    events = server_service.list_events("dhcp4")
    assert len(events.items) > 0
    for event in events.items:
        text: str = event.text.strip()
        assert not (
            text.startswith("Communication with <daemon") and
            text.endswith("failed")
        )

    # Fetches the host reservations properly.
    hosts = server_service.list_hosts('192.0.2.42')
    assert hosts is not None
    assert len(hosts.items) == 1
    host = hosts.items[0]
    local_hosts = host["localHosts"]
    len(local_hosts) == 1
    local_host = local_hosts[0]
    assert local_host["dataSource"] == "api"
