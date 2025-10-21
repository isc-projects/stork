from playwright.sync_api import Page, expect
from tests.ui.playwright.pages.login_page import LoginPage
import os
import pytest


BASE_URL = os.getenv("STORK_BASE_URL", "http://localhost:42080")
ADMIN_USER = os.getenv("STORK_ADMIN_USER", "admin")
ADMIN_PASS = os.getenv("STORK_ADMIN_PASS", "admin")
NEW_ADMIN_PASS = os.getenv("STORK_NEW_PASS", "A123456a!")


@pytest.mark.usefixtures("clean_env")
@pytest.mark.ui
def test_login_happy_path(page: Page):
    lp = LoginPage(page)
    lp.open(BASE_URL)

    lp.login(ADMIN_USER, ADMIN_PASS)

    if lp.is_password_change_required():
        lp.change_password(ADMIN_PASS, NEW_ADMIN_PASS)
        expect(lp.toast_password_updated()).to_be_visible(timeout=5000)

    lp.await_dashboard()

    lp.logout()
    page.wait_for_url("**/login*", timeout=10_000)

    lp.login(ADMIN_USER, NEW_ADMIN_PASS)
    lp.await_dashboard()


@pytest.mark.usefixtures("clean_env")
@pytest.mark.ui
def test_negative_password_change_mismatch(page: Page):
    lp = LoginPage(page)
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, ADMIN_PASS)

    assert lp.is_password_change_required(), "Expected forced password-change dialog"

    lp.old_password().fill(ADMIN_PASS)
    lp.new_password().fill("A123456a!")
    lp.confirm_password().fill("A123456a!x")

    expect(lp.error_mismatch_confirm()).to_be_visible(timeout=3000)


@pytest.mark.ui
def test_negative_invalid_login(page: Page):
    lp = LoginPage(page)
    lp.open(BASE_URL)

    lp.login(ADMIN_USER, "not-the-right-password")

    expect(lp.toast_invalid_login()).to_be_visible(timeout=5000)
