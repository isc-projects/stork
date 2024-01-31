import pytest

from core.fixtures import kea_parametrize
from core.wrappers import Server, Kea


@pytest.mark.skip(
    reason="The test is unstable because it doesn't wait to finish loading "
    "the Kea configuration. Unlike other tests, the loading here takes "
    "too much time and crashes the execution."
)
@kea_parametrize("agent-kea-many-subnets")
def test_add_kea_with_many_subnets(server_service: Server, kea_service: Kea):
    """Check if Stork agent and server will handle Kea instance with huge amount of subnets."""
    server_service.log_in_as_admin()
    machines = server_service.authorize_all_machines()
    assert len(machines.items) == 1
    state, *_ = server_service.wait_for_next_machine_states()

    assert state.apps is not None
    assert len(state.apps) == 1
    assert len(state.apps[0].access_points) == 1
    assert state.apps[0].access_points[0].address == "127.0.0.1"

    subnets = server_service.list_subnets(family=4)
    assert subnets.total == 6912
