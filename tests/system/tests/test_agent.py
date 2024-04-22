from core.wrappers import Kea, Server
from core.fixtures import kea_parametrize


def test_agent_reregistration_after_restart(server_service: Server, kea_service: Kea):
    """Check if the agent doesn't re-register after restart.
    It should use the same agent token and certs as before restart."""
    server_service.log_in_as_admin()
    machine_before = server_service.authorize_all_machines().items[0]
    hashes_before = kea_service.hash_cert_files()

    kea_service.restart_stork_agent()

    machine_after = server_service.list_machines().items[0]
    hashes_after = kea_service.hash_cert_files()

    assert machine_before.agent_token == machine_after.agent_token
    assert hashes_before == hashes_after


@kea_parametrize("agent-kea6")
def test_agent_over_ipv6(server_service: Server, kea_service: Kea):
    server_service.log_in_as_admin()
    machine = server_service.authorize_all_machines().items[0]

    assert ":" in machine.address

    state = server_service.wait_for_next_machine_state(machine.id)

    assert len(state.apps) > 0


@kea_parametrize("agent-kea-tls-optional-client-cert-no-verify")
def test_communication_with_kea_over_secure_protocol(
    server_service: Server, kea_service: Kea
):
    """Check if Stork agent communicates with Kea over HTTPS correctly.
    In this test the Kea doesn't require SSL certificate on the client side."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert state.apps[0].access_points[0].use_secure_protocol
    leases = server_service.list_leases("192.0.2.1")
    assert leases.total == 1


@kea_parametrize("agent-kea-tls-required-client-cert-no-verify")
def test_communication_with_kea_over_secure_protocol_non_trusted_client(
    server_service: Server, kea_service: Kea
):
    """The Stork Agent uses self-signed TLS certificates over HTTPS, but the Kea
    requires the valid credentials. The Stork Agent should send request, but get
    rejection from the Kea CA."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert state.apps[0].access_points[0].use_secure_protocol
    leases = server_service.list_leases("192.0.2.1")
    assert leases.items is None
    assert kea_service.has_failed_tls_handshake_log_entry()


@kea_parametrize("agent-kea-tls-optional-client-cert-verify")
def test_communication_with_kea_over_secure_protocol_require_trusted_cert(
    server_service: Server, kea_service: Kea
):
    """Check if Stork agent requires a trusted Kea cert if specific flag is not set.
    In this test the Kea doesn't require TLS certificate on the client side."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert state.apps[0].access_points[0].use_secure_protocol
    leases = server_service.list_leases("192.0.2.1")
    assert leases.items is None
    assert kea_service.has_failed_tls_handshake_log_entry()


@kea_parametrize("agent-kea-basic-auth-no-credentials")
def test_communication_with_kea_using_basic_auth_no_credentials(
    server_service: Server, kea_service: Kea
):
    """The Kea CA is configured to accept requests only with Basic Auth credentials in header.
    The Stork Agent doesn't have a credentials file. Kea shouldn't accept the requests from the Stork Agent.
    """
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    # trig forward command to Kea
    server_service.wait_for_next_machine_states()
    # The Stork Agent doesn't know the credentials.
    # The above request should fail.
    server_service.wait_for_failed_ca_communication()


@kea_parametrize("agent-kea-basic-auth")
def test_communication_with_kea_using_basic_auth(
    server_service: Server, kea_service: Kea
):
    """The Kea CA is configured to accept requests only with Basic Auth credentials in header.
    The Stork Agent has a credentials file. Kea should accept the requests from the Stork Agent.
    """
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    # Check communication
    leases = server_service.list_leases("192.0.2.1")
    assert leases.total == 1


@kea_parametrize(suppress_registration=True)
def test_kea_integer_overflow_in_statistics(kea_service: Kea):
    """
    Kea from version ~2.3 no longer returns the negative number if the
    statistic value overflows the int64 range. Instead, it throws an error like
    below after receiving statistics-get-all command:

    > internal server error: unable to parse server's answer to the forwarded
    > message: Number overflow: 2417851639229258349412352 in <wire>:0:5379

    We cannot work around this issue in the Stork Agent. The Kea doesn't
    provide the possibility to omit the statistics that are too big.

    The big numbers support was added in Kea 2.5.3.
    """
    kea_service.wait_for_detect_kea_applications()
    kea_version = kea_service.get_version()
    metrics = kea_service.read_prometheus_metrics()

    if kea_version < (2, 3):
        assert len(metrics) > 0
        assert "kea_dhcp6_na_total" in metrics
        expected_nas = pow(2, 128 - 80) * 4 + (-1)
        assert sum(s.value for s in metrics["kea_dhcp6_na_total"].samples) == expected_nas
    elif kea_version >= (2, 3) and kea_version < (2, 5, 3):
        assert kea_service.has_number_overflow_log_entry()
        assert "kea_dhcp6_na_total" not in metrics
    else:
        assert len(metrics) > 0
        assert "kea_dhcp6_na_total" in metrics
        expected_nas = pow(2, 128 - 80) * 4 + pow(2, 128 - 48) * 2
        assert sum(s.value for s in metrics["kea_dhcp6_na_total"].samples) == expected_nas
