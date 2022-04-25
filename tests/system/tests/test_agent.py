from core.wrappers import Kea, Server
from core.fixtures import kea_parametrize


def test_agent_reregistration_after_restart(server_service: Server, kea_service: Kea):
    """Check if after restart the agent isn't re-register.
       It should use the same agent token and certs as before restart."""
    server_service.log_in_as_admin()
    machine_before = server_service.authorize_all_machines()['items'][0]
    hashes_before = kea_service.hash_cert_files()

    kea_service.restart_stork_agent()

    machine_after = server_service.list_machines()['items'][0]
    hashes_after = kea_service.hash_cert_files()

    assert machine_before["agentToken"] == machine_after['agentToken']
    assert hashes_before == hashes_after


@kea_parametrize("agent-kea6")
def test_agent_over_ipv6(server_service: Server, kea_service: Kea):
    server_service.log_in_as_admin()
    machine = server_service.authorize_all_machines()['items'][0]

    assert ":" in machine['address']

    state = server_service.wait_for_next_machine_state(machine['id'])

    assert len(state["apps"]) > 0


@kea_parametrize("agent-kea-tls-optional-client-cert-no-verify")
def test_communication_with_kea_over_secure_protocol(server_service: Server, kea_service: Kea):
    """Check if Stork agent communicates with Kea over HTTPS correctly.
    In this test the Kea doesn't require SSL certificate on the client side."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert state['apps'][0]['access_points'][0]['use_secure_protocol']
    leases = server_service.list_leases('192.0.2.1')
    assert leases['total'] == 1


@kea_parametrize("agent-kea-tls-required-client-cert-no-verify")
def test_communication_with_kea_over_secure_protocol_nontrusted_client(server_service: Server, kea_service: Kea):
    """The Stork Agent uses self-signed TLS certificates over HTTPS, but the Kea
    requires the valid credentials. The Stork Agent should send request, but get
    rejection from the Kea CA."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert state['apps'][0]['access_points'][0]['use_secure_protocol']
    leases = server_service.list_leases('192.0.2.1')
    assert leases['items'] is None
    assert kea_service.has_failed_TLS_handshake_log_entry()


@kea_parametrize("agent-kea-tls-optional-client-cert-verify")
def test_communication_with_kea_over_secure_protocol_require_trusted_cert(server_service: Server, kea_service: Kea):
    """Check if Stork agent requires a trusted Kea cert if specific flag is not set.
    In this test the Kea doesn't require TLS certificate on the client side."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert state['apps'][0]['access_points'][0]['use_secure_protocol']
    leases = server_service.list_leases('192.0.2.1')
    assert leases['items'] is None
    assert kea_service.has_failed_TLS_handshake_log_entry()


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
    data = server_service.list_config_reports(daemon_id)

    # Expecting one report indicating that the stat_cmds hooks library
    # was not loaded.
    assert data['total'] == 1
    assert len(data['items']) == 1
    assert data['items'][0]['checker'] == 'stat_cmds_presence'


@kea_parametrize("agent-kea-basic-auth-no-credentials")
def test_communication_with_kea_using_basic_auth_no_credentials(server_service: Server, kea_service: Kea):
    """The Kea CA is configured to accept requests only with Basic Auth credentials in header.
    The Stork Agent doesn't have a credentials file. Kea shouldn't accept the requests from the Stork Agent."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    # trig forward command to Kea
    server_service.wait_for_next_machine_states()
    # The Stork Agent doesn't know the credentials.
    # The above request should fail.
    server_service.wait_for_failed_CA_communication()


@kea_parametrize("agent-kea-basic-auth")
def test_communication_with_kea_using_basic_auth(server_service: Server, kea_service: Kea):
    """The Kea CA is configured to accept requests only with Basic Auth credentials in header.
    The Stork Agent has a credentials file. Kea should accept the requests from the Stork Agent."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    # Check communication
    leases = server_service.list_leases('192.0.2.1')
    assert leases['total'] == 1
