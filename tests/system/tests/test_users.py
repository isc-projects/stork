import pytest

from openapi_client.api.users_api import User
from openapi_client.exceptions import BadRequestException

from core.wrappers import Server


def test_users_management(server_service: Server):
    """Check if users can be fetched and added."""
    admin_user = server_service.log_in_as_admin()
    assert admin_user.login == "admin"

    users = server_service.list_users()
    assert users.total == 1
    assert "email" not in users.items[0]
    assert users.items[0].groups == [1]
    assert users.items[0].id == 1
    assert users.items[0].lastname == "admin"
    assert users.items[0].login == "admin"
    assert users.items[0].name == "admin"

    groups = server_service.list_groups()
    assert groups.total == 3
    assert len(groups.items) == 3
    assert groups.items[0].name in ["super-admin", "admin", "read-only"]
    assert groups.items[1].name in ["super-admin", "admin", "read-only"]
    assert groups.items[2].name in ["super-admin", "admin", "read-only"]

    server_service.create_user(
        User(
            id=0,
            login="user",
            email="a@example.org",
            name="John",
            lastname="Smith",
            groups=[],
            authentication_method_id="internal",
        ),
        "Password123!",
    )


def test_user_without_groups(server_service: Server):
    """Users with no assigned groups must be able to log in and out."""
    server_service.log_in_as_admin()
    server_service.create_user(
        User(
            id=0,
            login="user",
            email="a@example.org",
            name="John",
            lastname="Smith",
            groups=[],
            authentication_method_id="internal",
        ),
        "Password123!",
    )
    server_service.log_out()

    server_service.log_in("user", "Password123!")
    server_service.log_out()


def test_login_with_extremely_large_payload(server_service: Server):
    """
    It verifies that CVE-2025-8696 has been fixed.
    The bug was that the server accepted any arbitrary size of the request
    directed to the login form. The attacker could send a request so large that
    it would exhaust all available memory and cause the server to crash as a
    result.
    """
    # Generate big request. For testing purposes it doesn't need to extremely
    # huge because the server rejects now requests bigger than several
    # kilobytes.
    big_username = "a" * 10_000_000
    big_password = "b" * 10_000_000

    # TODO: Refactor the backend code to return HTTP 413 Payload Too Large.
    with pytest.raises(BadRequestException) as ex:
        # The Pydantic library validates the size of the request and rejects
        # sending such a big payload. We need to suppress it.
        server_service.log_in(
            big_username,
            big_password,
            suppress_client_side_validation=True,
        )
    assert "request body too large" in str(ex.value).lower()
