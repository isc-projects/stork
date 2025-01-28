from core.wrappers import Server, Kea
from core.fixtures import kea_parametrize


@kea_parametrize("agent-kea-premium-host-database")
def test_get_host_reservation_from_host_db(kea_service: Kea, server_service: Server):
    """Tests that the host reservations are fetched from the host database by
    the hosts_cmds hook."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_host_reservation_pulling()

    # List hosts
    hosts = server_service.list_hosts("192.0.2.42")
    assert hosts is not None
    assert len(hosts.items) == 1
    host = hosts.items[0]
    local_hosts = host.local_hosts
    assert len(local_hosts) == 1
    local_host = local_hosts[0]
    assert local_host.data_source == "api"


@kea_parametrize("agent-kea-premium-host-database")
def test_add_host_reservation(kea_service: Kea, server_service: Server):
    """Tests that the new host reservation is inserted properly."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_host_reservation_pulling()

    # Add host
    raw_host = {
        "host_identifiers": [
            {"id_type": "flex-id", "id_hex_value": "01:02:03:04:05:06"}
        ],
        "address_reservations": [{"address": "10.42.42.42"}],
        "hostname": "foobar",
        "local_hosts": [],
    }

    with server_service.transaction_create_host_reservation() as (ctx, submit, _):
        daemon = [d for d in ctx.daemons if d.name == "dhcp4"][0]
        raw_host["local_hosts"].append({"daemon_id": daemon.id})
        submit(raw_host)

    server_service.wait_for_host_reservation_pulling()
    hosts = server_service.list_hosts("10.42.42.42")
    assert hosts.total == 1
    host = hosts.items[0]
    assert host.hostname == "foobar"
    assert host.address_reservations[0].address == "10.42.42.42"
    identifier = host.host_identifiers[0]
    assert identifier.id_type == "flex-id"
    assert identifier.id_hex_value == "01:02:03:04:05:06"


@kea_parametrize("agent-kea-premium-host-database")
def test_add_host_reservation_with_dash_delimiter(
    kea_service: Kea, server_service: Server
):
    """Tests that the new host reservation is inserted properly even if the
    hex identifier has been provided with a dash delimiter."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_host_reservation_pulling()

    # Add host - dash delimiter
    raw_host = {
        "host_identifiers": [
            {"id_type": "flex-id", "id_hex_value": "01-02-03-04-05-06"}
        ],
        "address_reservations": [{"address": "10.42.42.42"}],
        "hostname": "foobar",
        "local_hosts": [],
    }

    with server_service.transaction_create_host_reservation() as (ctx, submit, _):
        daemon = [d for d in ctx.daemons if d.name == "dhcp4"][0]
        raw_host["local_hosts"].append({"daemon_id": daemon.id})
        submit(raw_host)

    server_service.wait_for_host_reservation_pulling()
    hosts = server_service.list_hosts("10.42.42.42")
    host = hosts.items[0]
    assert hosts.total == 1
    assert host.hostname == "foobar"
    assert host.address_reservations[0].address == "10.42.42.42"
    identifier = host.host_identifiers[0]
    assert identifier.id_type == "flex-id"
    assert identifier.id_hex_value == "01:02:03:04:05:06"


@kea_parametrize("agent-kea-premium-host-database")
def test_cancel_host_reservation_transaction(kea_service: Kea, server_service: Server):
    """Tests that the host reservation transactions are canceled properly."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_host_reservation_pulling()

    host_id = server_service.list_hosts("192.0.2.42").items[0].id

    # Only one transaction for a given user may exist. The transaction
    # recreation after canceling checks if the previous one was correctly
    # invalidated.
    for _ in range(2):
        with server_service.transaction_create_host_reservation() as (_, _, cancel):
            cancel()

        with server_service.transaction_update_host_reservation(host_id) as (
            _,
            _,
            cancel,
        ):
            cancel()


@kea_parametrize("agent-kea-premium-host-database")
def test_update_host_reservation(kea_service: Kea, server_service: Server):
    """Tests that the host reservation is updated properly."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_host_reservation_pulling()

    # Fetch existing host reservation.
    hosts = server_service.list_hosts("192.0.2.42")
    host = hosts.items[0]

    # Modify the host reservation.
    host.id = host.id
    host.hostname = "barfoo"
    host.host_identifiers[0].id_type = "client-id"
    host.host_identifiers[0].id_hex_value = "06:05:04:03:02:01"
    host.address_reservations[0].address = "192.0.2.24"

    # Apply changes.
    server_service.update_host_reservation(host)

    # Wait for refresh the host reservation data.
    server_service.wait_for_host_reservation_pulling()

    # Check if the old entry was deleted.
    hosts = server_service.list_hosts("192.0.2.42")
    assert hosts.items is None

    # Check if the modified entry was updated.
    hosts = server_service.list_hosts("192.0.2.24")
    assert hosts.total == 1
    host = hosts.items[0]
    assert host.hostname == "barfoo"
    assert host.address_reservations[0].address == "192.0.2.24"
    identifier = host.host_identifiers[0]
    assert identifier.id_type == "client-id"
    assert identifier.id_hex_value == "06:05:04:03:02:01"
