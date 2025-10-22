from playwright.sync_api import Page, expect
import re
import time


class UserManagementPage:
    """User management actions using the selectors/flows you provided."""

    def __init__(self, page: Page):
        self.page = page

    def open_users(self):

        self.page.get_by_role("button", name="Navigation").click()
        self.page.locator("a").filter(has_text="Configuration").click()
        self.page.locator("#users a").click()

    def create_user(
        self,
        login: str,
        first: str,
        last: str,
        role: str,
        password: str,
        force_change_password: bool = False,
    ):
        self.page.get_by_role("button", name="Create User Account").click()

        self.page.get_by_role("textbox", name="Login*:").fill(login)
        self.page.get_by_role("textbox", name="First name*:").fill(first)
        self.page.get_by_role("textbox", name="Last name*:").fill(last)

        self.page.get_by_role("button", name="dropdown trigger").click()
        self.page.get_by_role("option", name=role, exact=True).click()

        self.page.get_by_role("textbox", name="Password*:", exact=True).fill(password)
        self.page.get_by_role("textbox", name="Repeat password*:").fill(password)

        if not force_change_password:
            self.page.locator("[data-pc-name='checkbox']").first.click()

        self.page.get_by_role("button", name="Save").click()

    def configuration_has_users_entry(self, timeout_ms: int = 1500) -> bool:

        self.page.get_by_role("button", name="Navigation").click()
        self.page.locator("a").filter(has_text="Configuration").click()

        loc = self.page.locator("#users a")
        deadline = time.monotonic() + (timeout_ms / 1000.0)

        while time.monotonic() < deadline:
            if loc.is_visible():
                return True
            time.sleep(0.05)
        return False

    def open_profile(self):

        self.page.locator("#logout-button").get_by_role("button").filter(
            has_text=re.compile(r"^$")
        ).click()
        self.page.get_by_role("link", name="Profile").click()

    def total_users_should_be(self, count: int):
        suffix = "" if count == 1 else "s"
        expect(
            self.page.get_by_text(f"Total: {count} user{suffix}", exact=True)
        ).to_be_visible(timeout=3000)

    def user_should_be_listed(self, login: str):
        expect(self.page.get_by_role("cell", name=login, exact=True)).to_be_visible(
            timeout=3000
        )

    def delete_user(self, login: str):
        self.page.get_by_role("link", name=login, exact=True).click()
        self.page.get_by_role("button", name="Delete").click()
        self.page.get_by_role("button", name="Yes").click()
