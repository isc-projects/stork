from core.wrappers import Server, Bind9


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
