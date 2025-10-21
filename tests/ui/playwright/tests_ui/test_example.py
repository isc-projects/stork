from playwright.sync_api import Page
import pytest
from tests.ui.playwright.pages.login_page import LoginPage
from tests.ui.playwright.pages.navigation import Navigation
from tests.ui.playwright.pages.shared_network_page import SharedNetworkPage


@pytest.mark.skip(reason="temporarily disabled")
def test_shared_network_edit_bug(page: Page, base_url: str):
    login_page = LoginPage(page)
    navigation_page = Navigation(page)
    shared_page = SharedNetworkPage(page)

    # Open & login on the SAME stack the system tests start
    login_page.open(base_url)
    login_page.login("admin", "admin")

    navigation_page.go_to_shared_network("esperanto")

    shared_page.edit_network(valid_lifetime="50", min_valid_lifetime="100")
    shared_page.expect_failure_toast()

    shared_page.edit_network(min_valid_lifetime="40")
    shared_page.network_cell.wait_for()
    shared_page.open_shared_network()
