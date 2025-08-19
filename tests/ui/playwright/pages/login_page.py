"""Page object for the Stork login page."""

# pylint: disable=too-few-public-methods
from playwright.sync_api import Page


class LoginPage:
    """Encapsulates interactions with the login screen."""

    def __init__(self, page: Page):
        """Initialize the login page."""
        self.page = page

    def login(self, username: str, password: str):
        """Login to Stork."""
        self.page.goto("http://localhost:8080/login?returnUrl=%2Fdashboard")
        self.page.get_by_role("textbox", name="Email/Login").fill(username)
        self.page.locator("input[type='password']").fill(password)
        self.page.get_by_role("button", name="Sign In").click()
