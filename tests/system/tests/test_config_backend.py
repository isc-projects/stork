from core.wrappers import Server, Kea
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
def test_get_host_reservations_from_radius(kea_service: Kea, server_service: Server):
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_host_reservation_pulling()

    events = server_service.list_events("dhcp4")
    assert len(events) > 0
