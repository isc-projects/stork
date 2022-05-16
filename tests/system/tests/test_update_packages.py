from core.wrappers import ExternalPackages
from core.fixtures import external_parametrize
from core.version import get_version


@external_parametrize(version="1.2")
def test_update_stork(external_service: ExternalPackages):
    with external_service.no_validate() as legacy_service:
        legacy_service.log_in_as_admin()
        legacy_service.authorize_all_machines()
        state = legacy_service.wait_for_next_machine_states()[0]
        assert state["agent_version"] == "1.2.0"
        assert legacy_service.read_version()["version"] == "1.2.0"

    external_service.update_agent_to_latest_version()
    external_service.update_server_to_latest_version()

    state = external_service.wait_for_next_machine_states()[0]
    expected_version = get_version()
    assert state["agent_version"] == expected_version
    assert external_service.read_version()["version"] == expected_version
