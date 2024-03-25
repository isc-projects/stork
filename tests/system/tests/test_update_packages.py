import pytest

from core.wrappers import ExternalPackages
from core import version


@pytest.mark.skip(
    reason="This test takes too much time and resources to run on every commit"
    "on CI. It should be run on demand. Now, it is disabled "
    "permanently, but it should be handled using the profile feature, "
    "similar to the tests that use the premium features."
)
def test_update_stork_from_the_latest_released_version(
    package_service: ExternalPackages,
):
    """
    Initializes the Stork Server with the packages from the CloudSmith and
    installs current packages.
    """
    expected_version_info = version.get_version_info()
    package_service.log_in_as_admin()
    machine = package_service.authorize_all_machines().items[0]

    state = package_service.read_machine_state(machine.id)

    agent_version = version.parse_version_info(state.agent_version)
    server_version = version.parse_version_info(package_service.read_version().version)
    # We change the version in the release phase.
    # During the development the latest CloudSmith version equals to the
    # version in the GO files but during the release it is lower.
    assert agent_version <= expected_version_info
    assert server_version <= expected_version_info

    package_service.update_server_to_latest_version()
    package_service.update_agent_to_latest_version()

    state = package_service.wait_for_next_machine_states(wait_for_apps=False)[0]
    agent_version = version.parse_version_info(state.agent_version)
    server_version = version.parse_version_info(package_service.read_version().version)
    assert agent_version == expected_version_info
    assert server_version == expected_version_info
