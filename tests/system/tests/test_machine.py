from core.wrappers import Server, Kea
from core.fixtures import kea_parametrize


@kea_parametrize("agent-kea-config-review")
def test_delete_machine_with_config_reports(kea_service: Kea, server_service: Server):
    """Test that the authorized machine having some config reports (with and
    without issues) is removed properly."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()
    daemon = [d for d in state.apps[0].details.daemons if d.name == "dhcp4"][0]
    reports = server_service.wait_for_config_reports(daemon.id)
    assert reports.total != 0
    assert any(r for r in reports.items if r.content is not None)
    assert any(r for r in reports.items if r.content is None)

    server_service.delete_machine(state.id)
    machines = server_service.list_machines()
    assert machines.total is None
    assert len(machines.items) == 0
