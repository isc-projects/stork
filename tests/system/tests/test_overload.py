from typing import Tuple

import pytest

from core.fixtures import kea_parametrize, ha_pair_parametrize
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


@pytest.mark.skip(reason="The test is disabled because it consumes too much resources.")
@ha_pair_parametrize("agent-kea-many-subnets", "agent-kea-many-subnets2")
def test_two_same_big_configurations_at_time(
    server_service: Server, ha_pair_service: Tuple[Kea, Kea]
):
    """
    Test verifies if Stork server is operational even if two Kea instances
    with many shared networks including many subnets are running at the same
    time.

    The ha_pair_service is used to start two Kea instances simultaneously.
    They don't have HA configuration.
    """
    server_service.log_in_as_admin()
    machines = server_service.authorize_all_machines()
    assert len(machines.items) == 2
    state, *_ = server_service.wait_for_next_machine_states()
    assert state
    state, *_ = server_service.wait_for_next_machine_states()
    assert state
