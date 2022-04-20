from core.wrappers import Server, Bind


def test_bind9_versions(server_service: Server, bind_service: Bind):
    """Check if Stork agent detects different BIND 9 versions."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    state, *_ = server_service.wait_for_next_machine_states()

    assert len(state['apps']) == 1
    app = state['apps'][0]
    assert len(app['accessPoints']) == 2
    assert app['accessPoints'][0]['address'] == '127.0.0.1'
