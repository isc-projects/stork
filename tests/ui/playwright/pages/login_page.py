import re
from playwright.sync_api import Page, expect


class LoginPage:
    """Encapsulates interactions with the login screen."""

    def __init__(self, page: Page):
        self.page = page

    def open(self, base_url: str):

        self.page.goto(base_url, wait_until="domcontentloaded")

    def _username_locator(self):

        selector = (
            "input[type='email'], "
            "input[type='text'], "
            "input[formcontrolname*='login' i], "
            "input[placeholder*='login' i], "
            "input[placeholder*='email' i]"
        )
        return self.page.locator(selector).first

    def _password_locator(self):
        return self.page.locator("input[type='password']").first

    def login(self, username: str, password: str):
        self.page.wait_for_load_state("networkidle")

        user = self._username_locator()
        pwd = self._password_locator()
        expect(user).to_be_visible(timeout=15000)
        expect(pwd).to_be_visible(timeout=15000)

        user.fill(username)
        pwd.fill(password)

        btn = self.page.get_by_role(
            "button", name=re.compile(r"(sign in|log in|login)", re.I)
        )
        if not btn.count():
            btn = self.page.locator("button[type='submit']").first
        btn.click()

        self.page.wait_for_load_state("networkidle")
        try:
            expect(self._password_locator()).not_to_be_visible(timeout=10000)
        except Exception:
            self.page.wait_for_url("**/dashboard*", timeout=10000)
