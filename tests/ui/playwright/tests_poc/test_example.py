"""Example Playwright test for Stork UI."""

from playwright.sync_api import Page
from tests.ui.playwright.pages.login_page import LoginPage
from tests.ui.playwright.pages.navigation import Navigation
from tests.ui.playwright.pages.shared_network_page import SharedNetworkPage


def test_shared_network_edit_bug(page: Page):
    """Test the shared network edit bug."""
    login_page = LoginPage(page)
    navigation_page = Navigation(page)
    shared_page = SharedNetworkPage(page)

    page.goto("http://localhost:8080/login?returnUrl=%2Fdashboard")
    login_page.login("admin", "A123456a!")

    navigation_page.go_to_shared_network("esperanto")

    shared_page.edit_network(valid_lifetime="50", min_valid_lifetime="100")
    shared_page.expect_failure_toast()

    shared_page.edit_network(min_valid_lifetime="40")
    # Expect the network to be visible without manual refresh
    shared_page.network_cell.wait_for()
    shared_page.open_shared_network()
