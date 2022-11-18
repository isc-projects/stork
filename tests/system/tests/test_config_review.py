from core.fixtures import kea_parametrize
from core.wrappers import Server, Kea


@kea_parametrize("agent-kea-config-review")
def test_get_dhcp4_config_review_reports(server_service: Server, kea_service: Kea):
    """Test that the Stork server performs Kea configuration review and returns the reports."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    daemons = state['apps'][0]['details']['daemons']
    daemons = [d for d in daemons if d['name'] == 'dhcp4']
    assert len(daemons) == 1
    daemon_id = daemons[0]['id']

    # Get config reports for the daemon.
    data = server_service.wait_for_config_reports(daemon_id)

    assert data['total'] > 3
    issue_reports = {}
    for report in data['items']:
        if 'content' in report:
            issue_reports[report['checker']] = report

    assert len(issue_reports) == 3

    assert 'stat_cmds_presence' in issue_reports
    assert 'overlapping_subnet' in issue_reports
    assert 'canonical_prefix' in issue_reports
