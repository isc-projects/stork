import os
import pytest
from tests.ui.playwright.pages.login_page import LoginPage
from tests.ui.playwright.pages.machines import MachinesPage

BASE_URL = os.getenv("STORK_BASE_URL", "http://localhost:42080")
ADMIN_USER = os.getenv("STORK_ADMIN_USER", "admin")
ADMIN_PASS = os.getenv("STORK_ADMIN_PASS", "admin")
NEW_ADMIN_PASS = os.getenv("STORK_NEW_PASS", "A123456a!")


@pytest.mark.ui
@pytest.mark.usefixtures("clean_env")
def test_machines_unauthorized_to_authorized_flow(page):
    lp = LoginPage(page)
    mp = MachinesPage(page)

    # Login
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()

    # Navigate to Machines
    mp.open()

    # Go to Unauthorized
    mp.switch_to_unauthorized()

    # Negative search
    mp.search("182")
    mp.expect_no_results_row()
    mp.click_clear_in_no_results_row()

    # Positive search (present)
    mp.search("172")
    mp.select_machine_row("172.42.42.100:8080")

    # Authorize
    mp.authorize_selected()

    # Switch to Authorized, clear and refresh to match your steps
    mp.switch_to_authorized()
    mp.clear_filters()
    mp.refresh_list()

    # Negative search
    mp.search("negativetest")
    mp.expect_no_results_row()
    mp.click_clear_in_no_results_row()

    # Search actual authorized machine and open it
    mp.search("agent")
    mp.open_machine("agent-kea")

    # Verify elements on the Machines details page
    mp.expect_detail_headings()
    mp.expect_detail_ip_fragment("172.42.42.100:")

    # Get latest state
    mp.get_latest_state()

    # Back and clear filters
    mp.back_to_machines_tab()
    mp.clear_filters()

    # Logout
    lp.logout("admin")


@pytest.mark.ui
@pytest.mark.usefixtures("clean_env")
def test_machines_authorize_via_actions_and_cleanup(page):
    lp = LoginPage(page)
    mp = MachinesPage(page)

    # Login
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()

    # Navigate: Navigation → Services → Machines → Unauthorized
    mp.open()
    mp.switch_to_unauthorized()

    row_key = "172.42.42.100:8080"

    # Select row and authorize via row-scoped Actions menu
    mp.select_machine_row(row_key)
    mp.wait_for_row(row_key)
    mp.open_actions_menu()
    mp.actions_authorize_from_menu()

    # Switch to Authorized and perform actions: refresh, download, then remove
    mp.switch_to_authorized()
    mp.wait_for_row(row_key)

    mp.open_actions_menu()
    mp.actions_refresh_state_from_menu()

    mp.open_actions_menu()
    mp.actions_download_archive_from_menu()

    mp.open_actions_menu()
    mp.actions_remove_machine_from_menu()

    # Logout
    lp.logout("admin")


@pytest.mark.ui
def test_machines_installing_agent_dialog(page):
    lp = LoginPage(page)
    mp = MachinesPage(page)

    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()
    mp.open()

    # Open install dialog
    mp.open_install_dialog()
    mp.expect_install_dialog_title()

    mp.assert_docs_link_opens_new_tab()

    # Verify the command snippet, try the copy button, regenerate token, then close
    mp.expect_wget_snippet_visible()
    mp.click_copy_first()
    mp.regenerate_token_and_wait()
    mp.close_install_dialog()

    # Logout
    lp.logout("admin")
