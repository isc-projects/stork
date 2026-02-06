from core.wrappers import Server, Kea
from core.fixtures import kea_parametrize
from openapi_client.models.kea_daemon import KeaDaemon


@kea_parametrize("agent-kea-config-review")
def test_delete_machine_with_config_reports(kea_service: Kea, server_service: Server):
    """Test that the authorized machine having some config reports (with and
    without issues) is removed properly."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()
    daemon = [d for d in state.daemons if d.name == "dhcp4"][0]
    reports = server_service.wait_for_config_reports(daemon.id)
    assert reports.total != 0
    assert any(r for r in reports.items if r.content is not None)
    assert any(r for r in reports.items if r.content is None)

    server_service.delete_machine(state.id)
    machines = server_service.list_machines()
    assert machines.total is None
    assert len(machines.items) == 0


@kea_parametrize("agent-kea-premium-host-database")
def test_fetch_machine_state(kea_service: Kea, server_service: Server):
    """Test that the machine state is fetched properly and all daemon info
    are stored in the database."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    machine, *_ = server_service.wait_for_next_machine_states()
    daemon: KeaDaemon
    dhcp4_daemons = [d for d in machine.daemons if d.name == "dhcp4"]
    assert len(dhcp4_daemons) == 1
    daemon = dhcp4_daemons[0]

    assert daemon.log_targets is not None
    assert len(daemon.log_targets) > 0
    assert daemon.hooks is not None
    assert len(daemon.hooks) > 0
    assert daemon.files is not None
    assert len(daemon.files) > 0
    assert daemon.backends is not None
    assert len(daemon.backends) > 0
