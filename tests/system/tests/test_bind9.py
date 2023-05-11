from core.wrappers import Server, Bind9
from core.fixtures import bind9_parametrize


def test_bind9(server_service: Server, bind9_service: Bind9):
    """Check if Stork Agent detects BIND 9."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert len(state['apps']) == 1
    app = state['apps'][0]
    assert app['type'] == "bind9"
    assert len(app['access_points']) == 2
    assert app['access_points'][0]['address'] == '127.0.0.1'

    # BIND9 rejects every second POST request if it contains a non-empty body.
    # See: https://gitlab.isc.org/isc-projects/bind9/-/issues/3463
    # This loop checks if the Stork isn't affected.
    for _ in range(2):
        metrics = {
            metric.name: metric
            for metric
            in bind9_service.read_prometheus_metrics()
        }
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

    assert len(state['apps']) == 1
    app = state['apps'][0]
    assert app['type'] == "bind9"
    assert len(app['access_points']) == 2
    assert app['access_points'][0]['address'] == '127.0.0.1'

    metrics = bind9_service.read_prometheus_metrics()
    assert metrics is not None
    assert len(list(metrics)) > 0
