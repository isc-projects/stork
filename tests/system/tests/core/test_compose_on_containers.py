from core.compose_factory import create_docker_compose


def test_fetch_empty_logs():
    compose = create_docker_compose()
    stdout, stderr = compose.logs()
    assert stderr == ""
    assert stdout != ""


def test_server_instance():
    service_name = "server"
    compose = create_docker_compose()
    compose.start(service_name)
    compose.wait_for_operational(service_name)
    state = compose.get_service_state(service_name)
    assert state.is_running()
    assert state.is_healthy()
    compose.stop()


def test_kea_only_instance():
    service_name = "agent-kea"
    env_vars = {"STORK_SERVER_URL": ""}
    compose = create_docker_compose(env_vars=env_vars)
    compose.start(service_name)
    compose.wait_for_operational(service_name)
    state = compose.get_service_state(service_name)
    assert state.is_running()
    assert state.is_healthy()
    compose.stop()
