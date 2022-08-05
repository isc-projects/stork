from core.wrappers import ExternalPackages
import core.version as version


def test_update_stork_from_the_latest_released_version(external_service: ExternalPackages):
    """
    Initializes the Stork Server with the packages from the CloudSmith and
    next install the current packages.
    """
    expected_version_info = version.get_version_info()

    with external_service.no_validate() as legacy_service:
        legacy_service.log_in_as_admin()
        legacy_service.authorize_all_machines()
        state = legacy_service.wait_for_next_machine_states(
            wait_for_apps=False
        )[0]
        agent_version = version.parse_version_info(state["agent_version"])
        server_version = version.parse_version_info(
            legacy_service.read_version()["version"])
        # We change the version in the release phase.
        # During the development the latest CloudSmith version equals to the
        # version in the GO files but during the release it is lower.
        assert agent_version <= expected_version_info
        assert server_version <= expected_version_info

    external_service.update_agent_to_latest_version()
    external_service.update_server_to_latest_version()

    state = external_service.wait_for_next_machine_states(
        wait_for_apps=False
    )[0]
    agent_version = version.parse_version_info(state["agent_version"])
    server_version = version.parse_version_info(
        external_service.read_version()["version"])
    assert agent_version == expected_version_info
    assert server_version == expected_version_info
