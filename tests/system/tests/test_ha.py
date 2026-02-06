from typing import Tuple
import warnings

from core.wrappers import Server, Kea


def test_ha_get_config_review_reports(
    server_service: Server, ha_service: Tuple[Kea, Kea, Kea]
):
    """Test that the HA peer works properly in the system tests."""
    server_service.log_in_as_admin()
    server_service.authorize_all_machines()
    server_service.wait_for_next_machine_states()
    server_service.wait_for_ha_ready()

    overview = server_service.overview()

    # Require 3 servers, each with 2 daemons (DHCPv4 and DHCPv6).
    assert len(overview.dhcp_daemons) >= 6
    if len(overview.dhcp_daemons) > 6:
        warnings.warn(
            f"Expected 6 daemons, but got {len(overview.dhcp_daemons)}. It "
            "means the race condition occurred on the detection phase. See #583."
        )

    assert all(getattr(d, "ha_enabled") for d in overview.dhcp_daemons)

    server_service.wait_for_ha_pulling()
    assert not server_service.has_deadlock_log_entry()
