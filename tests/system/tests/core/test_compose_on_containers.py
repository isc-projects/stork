from core.compose_factory import create_docker_compose


def test_server_instance():
    service_name = "server"
    compose = create_docker_compose()
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)
    state = compose.get_service_state(service_name)
    assert state.is_running()
    assert state.is_healthy()
    compose.down()


def test_kea_only_instance():
    service_name = "agent-kea"
    env_vars = {"STORK_SERVER_URL": ""}
    compose = create_docker_compose(extra_env_vars=env_vars)
    compose.bootstrap(service_name)
    compose.wait_for_operational(service_name)
    state = compose.get_service_state(service_name)
    assert state.is_running()
    assert state.is_healthy()
    compose.down()
