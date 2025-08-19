"""Navigation helpers for Stork UI flows."""

# pylint: disable=too-few-public-methods
import re
from playwright.sync_api import Page


class Navigation:
    """Encapsulates interactions with the Stork navigation."""

    def __init__(self, page: Page):
        """Initialize the navigation."""
        self.page = page

    def go_to_shared_network(self, name: str):
        """Navigate to a shared network."""
        self.page.get_by_role("button", name="Navigation").click()
        self.page.locator("div").filter(has_text=re.compile(r"^Services$")).locator(
            "a"
        ).click()
        self.page.get_by_role("link", name=" Machines").click()
        self.page.get_by_role("radio", name="Unauthorized").click()
        self.page.get_by_role("link", name="agent-kea-premium-one").click()
        self.page.locator("#show-machines-menu-1").click()
        self.page.get_by_role("menuitem", name="Authorize").locator("a").click()
        self.page.get_by_role("radio", name="Authorized", exact=True).click()
        self.page.get_by_role("link", name=re.compile(r"agent-kea")).click()
        self.page.get_by_role("link", name=" DHCPv4").click()
        self.page.get_by_role("button", name="Shared Networks").click()
        self.page.get_by_role("link", name=name).click()
