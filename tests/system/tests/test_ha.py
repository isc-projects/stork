from typing import Tuple

from core.wrappers import Server, Kea


def test_get_ha_config_review_reports(
    server_service: Server, ha_service: Tuple[Kea, Kea, Kea]
):
    """Test that the HA peer works properly in the system tests."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_ha_ready()

    overview = server_service.overview()

    # Require 3 servers, each with 2 daemons (DHCPv4 and DHCPv6).
    assert len(overview.dhcp_daemons) == 6
    assert all(getattr(d, "ha_enabled") for d in overview.dhcp_daemons)
