"""Page object for Shared Network screens and interactions."""

# pylint: disable=too-many-instance-attributes
# pylint: disable=line-too-long
from playwright.sync_api import Page


class SharedNetworkPage:
    """Encapsulates interactions with the Shared Network screens."""

    def __init__(self, page: Page):
        self.page = page
        self.shared_network_button = page.get_by_role("button", name="Shared Networks")
        self.network_link = page.get_by_role("link", name="esperanto")
        self.edit_button = page.get_by_role("button", name="Edit")
        self.valid_lifetime_field = page.locator(
            ".p-inputtext.p-component.p-element.p-inputnumber-input.p-filled"
        ).first
        self.min_valid_lifetime_field = page.locator(
            ".p-element.p-inputwrapper.max-w-form.p-inputwrapper-filled.ng-untouched > .p-inputnumber > .p-inputtext"
        ).first
        self.submit_button = page.get_by_role("button", name="Submit")
        self.refresh_button = page.get_by_role("button", name=" Refresh List")
        self.no_networks_cell = page.get_by_role(
            "cell", name="No shared networks found."
        )
        self.network_cell = page.get_by_role("cell", name="esperanto")
        self.toast_error = page.get_by_text("Failed to update the shared")
        self.toast_commit_error = page.get_by_text("Cannot commit shared network")

    def open_shared_network(self):
        """Open the shared network."""
        self.shared_network_button.click()
        self.network_link.click()

    def edit_network(self, valid_lifetime: str = "", min_valid_lifetime: str = ""):
        """Edit the shared network."""
        self.edit_button.click()
        if valid_lifetime:
            self.valid_lifetime_field().fill(valid_lifetime)
        if min_valid_lifetime:
            self.min_valid_lifetime_field().fill(min_valid_lifetime)
        self.submit_button.click()

    def expect_failure_toast(self):
        """Expect a failure toast."""
        self.toast_error.wait_for()
        self.toast_commit_error.wait_for()

    def refresh_networks(self):
        """Refresh the networks."""
        self.refresh_button.click()
