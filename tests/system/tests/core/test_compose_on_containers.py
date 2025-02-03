import hashlib
import os

from core.compose_factory import create_docker_compose
from core.constants import config_directory


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

    # Check if the Kea configuration is an isolated volume.
    config_path = os.path.join(config_directory, "kea", "kea-dhcp4.conf")
    with open(config_path, "rb") as f:
        content = f.read()
        h = hashlib.blake2b()
        h.update(content)
        hash_before = h.digest()

    # Modify the configuration file.
    compose.exec(
        service_name,
        ["sh", "-c", "echo '// foo' >> /etc/kea/kea-dhcp4.conf"],
    )

    # Check if the configuration file on host is still the same.
    with open(config_path, "rb") as f:
        content = f.read()
        h = hashlib.blake2b()
        h.update(content)
        hash_after = h.digest()

    assert hash_before == hash_after

    # Check if the isolated directory is created.
    isolated_directory = os.path.join(config_directory, ".isolated")
    assert os.path.exists(isolated_directory)

    compose.down()

    # Check if the isolated directory is removed.
    assert not os.path.exists(isolated_directory)
