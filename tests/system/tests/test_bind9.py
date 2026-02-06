import warnings

from core.wrappers import Server, Bind9
from core.fixtures import bind9_parametrize


def assertSingleDaemon(state):
    '''
    It is possible that the race condition occurs on the detection phase
    because the first state puller iteration may still be executed while the
    test fetches the state. This problem is described in #583.
    '''
    assert len(state.daemons) >= 1
    if len(state.daemons) > 1:
        warnings.warn(
            f"Expected 1 daemon, but got {len(state.daemons)}. It means the "
            "race condition occurred on the detection phase. See #583."
        )

def test_bind9(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent detects BIND 9."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"
    assert len(daemon.access_points) == 2
    assert daemon.access_points[0].address == "127.0.0.1"

    # BIND9 rejects every second POST request if it contains a non-empty body.
    # See: https://gitlab.isc.org/isc-projects/bind9/-/issues/3463
    # This loop checks if the Stork isn't affected.
    for _ in range(2):
        metrics = bind9_service.read_prometheus_metrics()
        up_metric = metrics["bind_up"]
        up_metric_value = up_metric.samples[0].value
        assert up_metric_value == 1.0


@bind9_parametrize("agent-bind9-rndc")
def test_bind9_rndc(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent can communicate with BIND 9 while RNDC
    authentication is enabled."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"
    assert len(daemon.access_points) == 2
    assert daemon.access_points[0].address == "127.0.0.1"

    metrics = bind9_service.read_prometheus_metrics()
    assert metrics is not None
    assert len(metrics) > 0


@bind9_parametrize("agent-bind9-package")
def test_bind9_package(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent can communicate with BIND 9 using an initial
    configuration installed from BIND9 binary package."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"
    assert len(daemon.access_points) == 1  # Missing statistics
    assert daemon.access_points[0].address == "127.0.0.1"


@bind9_parametrize("agent-bind9-rndc-custom")
def test_bind9_rndc_custom(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent can communicate with BIND 9 when RNDC uses a custom
    configuration file and the required key isn't specified in the rndc.key
    file."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"
    assert len(daemon.access_points) == 2
    assert daemon.access_points[0].address == "127.0.0.1"

@bind9_parametrize("agent-bind9-chroot")
def test_bind9_chroot(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent can monitor BIND 9 running in the chroot
    environment."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()
    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"


@bind9_parametrize("agent-bind9-chroot-rndc-custom")
def test_bind9_chroot_rndc_custom(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent can communicate with BIND 9 running in the chroot
    environment when RNDC uses a custom configuration file and the required
    key isn't specified in the rndc.key file."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"
    assert len(daemon.access_points) == 2
    assert daemon.access_points[0].address == "127.0.0.1"

def test_bind9_fetch_zones(server_service: Server, bind9_service: Bind9):
    """Check if zones can be fetched from BIND9."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assertSingleDaemon(state)
    daemon = state.daemons[0]
    assert daemon.name == "named"
    assert len(daemon.access_points) == 2
    assert daemon.access_points[0].address == "127.0.0.1"

    zone_inventory_states = server_service.wait_for_fetch_zones()

    assert len(zone_inventory_states.items) == 1
    assert zone_inventory_states.items[0].status == "ok"

    zones = server_service.get_zones(0, 1000)
    assert len(zones.items) == 105
