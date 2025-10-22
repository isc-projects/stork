import os
import pytest
import re
from playwright.sync_api import expect
from tests.ui.playwright.pages.login_page import LoginPage
from tests.ui.playwright.pages.user_management import UserManagementPage

BASE_URL = os.getenv("STORK_BASE_URL", "http://localhost:42080")
ADMIN_USER = os.getenv("STORK_ADMIN_USER", "admin")
ADMIN_PASS = os.getenv("STORK_ADMIN_PASS", "admin")
NEW_ADMIN_PASS = os.getenv("STORK_NEW_PASS", "A123456a!")
MAIN_PASS = "A123456a!"
TEMP_PASS = "temptestpass123"


@pytest.mark.ui
def test_create_read_only_user_and_verify_role(page):

    lp = LoginPage(page)
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)

    lp.await_dashboard()

    um = UserManagementPage(page)
    um.open_users()

    login = "testuser_ro"
    um.create_user(
        login=login,
        first="Test",
        last="UserRO",
        role="read-only",
        password=NEW_ADMIN_PASS,
        force_change_password=False,
    )

    # Log out admin
    lp.logout("admin")

    # Log in as the new read-only user
    lp.open(BASE_URL)
    lp.login(login, NEW_ADMIN_PASS)
    lp.await_dashboard()

    # Verify: "Users" entry should NOT be visible under Configuration for read-only
    has_users = um.configuration_has_users_entry()
    assert not has_users, "read-only should not see 'Users' in Configuration"

    # Verify role via Profile is read-only
    um.open_profile()
    expect(page.get_by_text("read-only", exact=True)).to_be_visible(timeout=3000)

    # Log out testuser_ro
    lp.logout("testuser_ro")


@pytest.mark.ui
def test_create_admin_user_and_verify_role(page):
    lp = LoginPage(page)
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()

    um = UserManagementPage(page)
    um.open_users()

    login = "testuser_admin"
    um.create_user(
        login=login,
        first="Test",
        last="UserAdmin",
        role="admin",
        password=NEW_ADMIN_PASS,
        force_change_password=False,
    )

    lp.logout("admin")

    lp.open(BASE_URL)
    lp.login(login, NEW_ADMIN_PASS)
    lp.await_dashboard()

    has_users = um.configuration_has_users_entry()
    assert has_users is False, "admin should not see 'Users' in Configuration menu"

    um.open_profile()
    expect(page.get_by_text("admin", exact=True)).to_be_visible(timeout=3000)

    lp.logout("testuser_admin")


@pytest.mark.ui
def test_create_super_admin_and_create_read_only_user(page):
    lp = LoginPage(page)
    um = UserManagementPage(page)

    # 1) Admin logs in and creates a super-admin user
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()

    um.open_users()
    super_login = "testuser_superadmin"
    um.create_user(
        login=super_login,
        first="Test",
        last="UserSuper",
        role="super-admin",
        password=NEW_ADMIN_PASS,
        force_change_password=False,
    )

    lp.logout("admin")

    # 2) Log in as the new super-admin
    lp.open(BASE_URL)
    lp.login(super_login, NEW_ADMIN_PASS)
    lp.await_dashboard()

    # Verify role via Profile is super-admin
    um.open_profile()
    expect(page.get_by_text("superadmin", exact=True)).to_be_visible(timeout=3000)

    # 3) As super-admin, create a new read-only user
    um.open_users()
    ro_login_2 = "testuser_ro2"
    um.create_user(
        login=ro_login_2,
        first="Test",
        last="UserRO2",
        role="read-only",
        password=NEW_ADMIN_PASS,
        force_change_password=False,
    )

    lp.logout("testuser_superadmin")

    lp.open(BASE_URL)
    lp.login(ro_login_2, NEW_ADMIN_PASS)
    lp.await_dashboard()

    # Verify Users entry is not visible in Configuration menu
    has_users = um.configuration_has_users_entry()
    assert (
        has_users is False
    ), "read-only user should not see 'Users' in Configuration menu"

    # Verify role via Profile is read-only
    um.open_profile()
    expect(page.get_by_text("read-only", exact=True)).to_be_visible(timeout=3000)

    lp.logout("testuser_ro2")


@pytest.mark.ui
def test_delete_created_users_and_verify_totals(page):
    lp = LoginPage(page)
    um = UserManagementPage(page)

    # Login as admin
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()

    # Navigate to Users
    um.open_users()

    # Verify total is 5 and the four new users are visible
    um.total_users_should_be(5)
    users = ["testuser_ro", "testuser_admin", "testuser_superadmin", "testuser_ro2"]
    for u in users:
        um.user_should_be_listed(u)

    # Delete each new user
    for u in users:
        um.delete_user(u)

    # Verify only 1 user remains
    um.total_users_should_be(1)

    # Logout
    lp.logout("admin")


@pytest.mark.ui
def test_delete_main_admin_shows_error(page):
    lp = LoginPage(page)
    um = UserManagementPage(page)

    # Login as main admin
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, NEW_ADMIN_PASS if NEW_ADMIN_PASS else ADMIN_PASS)
    lp.await_dashboard()

    # Go to Users
    um.open_users()

    # Attempt to delete the main admin user
    page.get_by_role("link", name="admin", exact=True).click()
    page.get_by_role("button", name="Delete").click()
    page.get_by_role("button", name="Yes").click()

    # Verify the error message
    expect(page.get_by_text("Failed to delete user account", exact=True)).to_be_visible(
        timeout=5000
    )

    # Logout
    lp.logout("admin")


@pytest.mark.ui
def test_change_admin_password_then_revert(page):
    lp = LoginPage(page)

    # 1) Login as admin with MAIN_PASS
    lp.open(BASE_URL)
    lp.login(ADMIN_USER, MAIN_PASS)
    lp.await_dashboard()

    # 2) Open Profile → Change password
    page.locator("#logout-button").get_by_role("button").filter(
        has_text=re.compile(r"^$")
    ).click()
    page.get_by_role("link", name="Profile").click()
    page.get_by_role("link", name=" Change password", exact=True).click()

    # 3) Change password: MAIN_PASS -> TEMP_PASS
    lp.old_password().fill(MAIN_PASS)
    lp.new_password().fill(TEMP_PASS)
    lp.confirm_password().fill(TEMP_PASS)
    lp.save_new_password_button().click()

    # 4) Logout
    lp.logout("admin")

    # 5) Try login with MAIN_PASS to confirm it fails
    page.get_by_role("textbox", name="Email/Login").fill(ADMIN_USER)
    page.locator("input[type='password']").fill(MAIN_PASS)
    page.get_by_role("button", name="Sign In").click()
    expect(page.get_by_text("Invalid login or password", exact=True)).to_be_visible(
        timeout=5000
    )

    # 6) Login with TEMP_PASS and land on dashboard
    page.locator("input[type='password']").press("ControlOrMeta+a")
    page.locator("input[type='password']").fill(TEMP_PASS)
    page.get_by_role("button", name="Sign In").click()
    lp.await_dashboard()

    # 7) Change password back: TEMP_PASS -> MAIN_PASS
    page.locator("#logout-button").get_by_role("button").filter(
        has_text=re.compile(r"^$")
    ).click()
    page.get_by_role("link", name="Profile").click()
    page.get_by_role("link", name=" Change password", exact=True).click()

    lp.old_password().fill(TEMP_PASS)
    lp.new_password().fill(MAIN_PASS)
    lp.confirm_password().fill(MAIN_PASS)
    lp.save_new_password_button().click()

    # 8) Logout and verify login with MAIN_PASS works again
    lp.logout("admin")
    page.get_by_role("textbox", name="Email/Login").fill(ADMIN_USER)
    page.locator("input[type='password']").fill(MAIN_PASS)
    page.get_by_role("button", name="Sign In").click()
    lp.await_dashboard()

    # Final logout
    lp.logout("admin")
